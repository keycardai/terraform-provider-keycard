// Command sweep deletes zones that acceptance tests left behind in the
// organization the KEYCARD_* credentials belong to.
//
// A test leaves its zone behind when it never reaches its own destroy: a go
// test timeout panics the whole binary, a cancelled CI job kills it, and a
// failed child delete stops Terraform before it gets to the zone. Deleting a
// zone deletes everything in it, so zones are the only thing swept.
//
// Usage:
//
//	go run ./cmd/sweep -dry-run   # list the zones that would be deleted
//	go run ./cmd/sweep            # delete them
//
// It needs KEYCARD_CLIENT_ID, KEYCARD_CLIENT_SECRET and KEYCARD_ENDPOINT.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/oauth2"

	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

const (
	// zonePrefix matches the names acceptance tests give their zones.
	// acctest.RandomWithPrefix("tftest...") always writes "<prefix>-<number>",
	// so every test zone name starts with "tftest-".
	zonePrefix = "tftest-"

	// minZoneAge keeps the sweeper away from zones a running test still owns.
	// It is well above the acceptance job timeout, so any older zone belongs
	// to a run that has already ended.
	minZoneAge = time.Hour

	// concurrency bounds parallel deletes. Each delete deregisters the zone
	// from Vault and the router, so a large backlog is slow serially.
	concurrency = 5

	// requestTimeout replaces the provider client's 5s timeout. A zone delete
	// calls Vault and the router, which can take longer under load.
	requestTimeout = 30 * time.Second

	// attempts bounds retries of a list page or a delete after a transport
	// error, a 429 or a 5xx.
	attempts = 4

	// pageSize is the largest page the zones API returns.
	pageSize = 100

	// maxPages stops a listing whose cursor never runs out.
	maxPages = 1000
)

// initialBackoff is the wait before the first retry; it doubles after each
// attempt. A variable so unit tests can shorten it.
var initialBackoff = 2 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.SetFlags(log.LstdFlags | log.LUTC)
	if err := run(ctx, os.Args[1:], os.Stderr); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

// run parses the flags and sweeps. Flag errors, such as -dry-run=yes, are
// returned before any credentials are read or requests are made.
func run(ctx context.Context, args []string, output io.Writer) error {
	fs := flag.NewFlagSet("sweep", flag.ContinueOnError)
	fs.SetOutput(output)
	dryRun := fs.Bool("dry-run", false, "list the zones that would be deleted, without deleting them")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	c, err := newClient(ctx)
	if err != nil {
		return err
	}
	return sweep(ctx, c, *dryRun)
}

// newClient builds an API client like client.NewAPIClient, but with a timeout
// long enough for zone deletes.
func newClient(ctx context.Context) (*client.ClientWithResponses, error) {
	for _, envVar := range []string{"KEYCARD_CLIENT_ID", "KEYCARD_CLIENT_SECRET", "KEYCARD_ENDPOINT"} {
		if os.Getenv(envVar) == "" {
			return nil, fmt.Errorf("%s must be set", envVar)
		}
	}

	endpoint := os.Getenv("KEYCARD_ENDPOINT")
	httpClient := oauth2.NewClient(ctx, client.NewTokenSource(os.Getenv("KEYCARD_CLIENT_ID"), os.Getenv("KEYCARD_CLIENT_SECRET"), endpoint))
	httpClient.Timeout = requestTimeout

	c, err := client.NewClientWithResponses(endpoint, client.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("creating API client: %w", err)
	}
	return c, nil
}

// sweep deletes the leftover test zones, or only lists them when dryRun is set.
func sweep(ctx context.Context, c *client.ClientWithResponses, dryRun bool) error {
	zones, err := listZones(ctx, c)
	if err != nil {
		return err
	}

	now := time.Now()
	var stale []client.Zone
	for _, zone := range zones {
		if sweepable(zone, now) {
			stale = append(stale, zone)
		}
	}

	if dryRun {
		for _, zone := range stale {
			log.Printf("Would delete zone %s (%s), created %s", zone.Id, zone.Name, zone.CreatedAt.Format(time.RFC3339))
		}
		log.Printf("Dry run: would delete %d of %d zones (name prefix %q, older than %s); nothing was deleted", len(stale), len(zones), zonePrefix, minZoneAge)
		return nil
	}

	log.Printf("Deleting %d of %d zones (name prefix %q, older than %s)", len(stale), len(zones), zonePrefix, minZoneAge)

	var (
		mu                   sync.Mutex
		errs                 []error
		deleted, alreadyGone int
		wg                   sync.WaitGroup
		sem                  = make(chan struct{}, concurrency)
	)
	for _, zone := range stale {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			gone, err := deleteZone(ctx, c, zone)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err != nil:
				errs = append(errs, err)
			case gone:
				alreadyGone++
			default:
				deleted++
			}
		}()
	}
	wg.Wait()

	log.Printf("Deleted %d zones, %d already gone, %d failed", deleted, alreadyGone, len(errs))
	return errors.Join(errs...)
}

// listZones returns every zone in the organization. It lists everything
// before anything is deleted, so deletes can't shift the cursor.
func listZones(ctx context.Context, c *client.ClientWithResponses) ([]client.Zone, error) {
	var zones []client.Zone
	var after *string
	limit := pageSize
	for range maxPages {
		resp, err := withRetry(ctx, func() (*client.ListZonesResponse, error) {
			return c.ListZonesWithResponse(ctx, &client.ListZonesParams{After: after, Limit: &limit})
		})
		if err != nil {
			return nil, fmt.Errorf("listing zones: %w", err)
		}
		if resp.JSON200 == nil {
			return nil, fmt.Errorf("listing zones, got status %d: %s", resp.StatusCode(), string(resp.Body))
		}
		zones = append(zones, resp.JSON200.Items...)

		cursor := resp.JSON200.Pagination.AfterCursor
		if !cursor.IsSpecified() || cursor.IsNull() || cursor.MustGet() == "" {
			return zones, nil
		}
		next := cursor.MustGet()
		after = &next
	}
	return nil, fmt.Errorf("listing zones: still more pages after %d; stopping", maxPages)
}

// sweepable reports whether a zone was left behind by an acceptance test that
// has finished.
func sweepable(zone client.Zone, now time.Time) bool {
	return strings.HasPrefix(zone.Name, zonePrefix) && now.Sub(zone.CreatedAt) >= minZoneAge
}

// deleteZone deletes a zone. It reports alreadyGone when the zone no longer
// existed, which is the outcome the sweeper wants either way.
func deleteZone(ctx context.Context, c *client.ClientWithResponses, zone client.Zone) (alreadyGone bool, err error) {
	resp, err := withRetry(ctx, func() (*client.DeleteZoneResponse, error) {
		return c.DeleteZoneWithResponse(ctx, zone.Id)
	})
	if err != nil {
		return false, fmt.Errorf("deleting zone %s (%s): %w", zone.Id, zone.Name, err)
	}
	switch resp.StatusCode() {
	case http.StatusNoContent:
		return false, nil
	case http.StatusNotFound:
		return true, nil
	default:
		return false, fmt.Errorf("deleting zone %s (%s), got status %d: %s", zone.Id, zone.Name, resp.StatusCode(), string(resp.Body))
	}
}

// apiResponse is implemented by every generated *WithResponse return type.
type apiResponse interface {
	StatusCode() int
}

// retryable reports whether a request should be tried again. A transport
// error is retried too: list is a read and DELETE is idempotent, so a delete
// that timed out but finished on the server returns 404 on the next attempt.
func retryable[R apiResponse](resp R, err error) bool {
	if err != nil {
		return true
	}
	status := resp.StatusCode()
	return status == http.StatusTooManyRequests || status >= 500
}

// withRetry calls call until it succeeds, returns a non-retryable response,
// or runs out of attempts, backing off exponentially in between.
func withRetry[R apiResponse](ctx context.Context, call func() (R, error)) (R, error) {
	delay := initialBackoff
	for attempt := 1; ; attempt++ {
		resp, err := call()
		if attempt == attempts || !retryable(resp, err) {
			return resp, err
		}

		select {
		case <-ctx.Done():
			return resp, ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
	}
}
