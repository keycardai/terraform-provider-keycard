package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/oapi-codegen/nullable"
	"golang.org/x/oauth2"

	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

// Sweepers delete what acceptance tests leave behind when they never reach
// their own destroy: a go test timeout panics the whole binary, a cancelled CI
// job kills it, and a failed child delete stops Terraform before it gets to the
// zone. Deleting a zone deletes everything in it, so zones are the only thing
// swept. Run with `make sweep`, or `make sweep SWEEP_DRY_RUN=1` to only list what
// would be deleted.

const (
	// sweepZonePrefix matches the names acceptance tests give their zones.
	// acctest.RandomWithPrefix("tftest...") always writes "<prefix>-<number>",
	// so every test zone name starts with "tftest-".
	sweepZonePrefix = "tftest-"

	// sweepZoneMinAge keeps the sweeper away from zones a running test still
	// owns. It is well above the acceptance job timeout, so any older zone
	// belongs to a run that has already ended.
	sweepZoneMinAge = time.Hour

	// sweepConcurrency bounds parallel deletes. Each delete deregisters the
	// zone from Vault and the router, so a large backlog is slow serially.
	sweepConcurrency = 5

	// sweepRequestTimeout replaces the provider client's 5s timeout. A zone
	// delete calls Vault and the router, which can take longer under load.
	sweepRequestTimeout = 30 * time.Second

	// sweepAttempts bounds retries of a list page or a delete after a
	// transport error, a 429 or a 5xx.
	sweepAttempts = 4
)

// sweepInitialBackoff is the wait before the first retry; it doubles after
// each attempt. A variable so unit tests can shorten it.
var sweepInitialBackoff = 2 * time.Second

// sweepDryRunEnv, when true, makes the sweeper list the zones it would delete
// without deleting them. It is an environment variable rather than a test flag
// so `make sweep SWEEP_DRY_RUN=1` reaches it: make exports command-line variables to
// the recipe's environment.
const sweepDryRunEnv = "SWEEP_DRY_RUN"

// sweepDryRun reads sweepDryRunEnv. An unset or empty value means a real sweep.
// A value strconv.ParseBool doesn't accept is an error, so a typo such as
// SWEEP_DRY_RUN=yes stops the sweep instead of deleting.
func sweepDryRun() (bool, error) {
	v := os.Getenv(sweepDryRunEnv)
	if v == "" {
		return false, nil
	}
	dryRun, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s=%q is not a boolean; use 1 or true for a dry run", sweepDryRunEnv, v)
	}
	return dryRun, nil
}

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func init() {
	resource.AddTestSweepers("keycard_zone", &resource.Sweeper{
		Name: "keycard_zone",
		F:    sweepZones,
	})
}

// sweepableZone reports whether a zone was left behind by an acceptance test
// that has finished.
func sweepableZone(zone client.Zone, now time.Time) bool {
	return strings.HasPrefix(zone.Name, sweepZonePrefix) && now.Sub(zone.CreatedAt) >= sweepZoneMinAge
}

// sweepRetryable reports whether a sweeper request should be tried again. A
// transport error is retried too: DELETE is idempotent, and a delete that
// timed out but finished on the server returns 404 on the next attempt.
func sweepRetryable[R apiResponse](resp R, err error) bool {
	if err != nil {
		return true
	}
	status := resp.StatusCode()
	return status == http.StatusTooManyRequests || status >= 500
}

// withSweepRetry calls call until it succeeds, returns a non-retryable
// response, or runs out of attempts, backing off exponentially in between.
func withSweepRetry[R apiResponse](ctx context.Context, call func() (R, error)) (R, error) {
	delay := sweepInitialBackoff
	for attempt := 1; ; attempt++ {
		resp, err := call()
		if attempt == sweepAttempts || !sweepRetryable(resp, err) {
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

// newSweepClient builds an API client like client.NewAPIClient, but with a
// timeout long enough for zone deletes.
func newSweepClient(ctx context.Context) (*client.ClientWithResponses, error) {
	for _, envVar := range []string{"KEYCARD_CLIENT_ID", "KEYCARD_CLIENT_SECRET", "KEYCARD_ENDPOINT"} {
		if os.Getenv(envVar) == "" {
			return nil, fmt.Errorf("%s must be set to run sweepers", envVar)
		}
	}

	endpoint := os.Getenv("KEYCARD_ENDPOINT")
	httpClient := oauth2.NewClient(ctx, client.NewTokenSource(os.Getenv("KEYCARD_CLIENT_ID"), os.Getenv("KEYCARD_CLIENT_SECRET"), endpoint))
	httpClient.Timeout = sweepRequestTimeout

	c, err := client.NewClientWithResponses(endpoint, client.WithHTTPClient(client.NewLoggingHTTPClient(httpClient)))
	if err != nil {
		return nil, fmt.Errorf("creating API client: %w", err)
	}
	return c, nil
}

// sweepZones deletes the leftover test zones in the organization the
// KEYCARD_* credentials belong to. The region argument is required by the
// sweeper framework and has no meaning here.
func sweepZones(_ string) error {
	dryRun, err := sweepDryRun()
	if err != nil {
		return err
	}

	ctx := context.Background()
	c, err := newSweepClient(ctx)
	if err != nil {
		return err
	}

	// List everything before deleting so deletes can't shift the cursor.
	limit := 100
	zones, err := listAllPages(func(after *string) ([]client.Zone, nullable.Nullable[string], error) {
		resp, err := withSweepRetry(ctx, func() (*client.ListZonesResponse, error) {
			return c.ListZonesWithResponse(ctx, &client.ListZonesParams{After: after, Limit: &limit})
		})
		if err != nil {
			return nil, nil, fmt.Errorf("listing zones: %w", err)
		}
		if resp.JSON200 == nil {
			return nil, nil, fmt.Errorf("listing zones, got status %d: %s", resp.StatusCode(), string(resp.Body))
		}
		return resp.JSON200.Items, resp.JSON200.Pagination.AfterCursor, nil
	})
	if err != nil {
		return err
	}

	now := time.Now()
	var stale []client.Zone
	for _, zone := range zones {
		if sweepableZone(zone, now) {
			stale = append(stale, zone)
		}
	}
	if dryRun {
		for _, zone := range stale {
			log.Printf("[INFO] Would delete zone %s (%s), created %s", zone.Id, zone.Name, zone.CreatedAt.Format(time.RFC3339))
		}
		log.Printf("[INFO] Dry run: would delete %d of %d zones (name prefix %q, older than %s); nothing was deleted", len(stale), len(zones), sweepZonePrefix, sweepZoneMinAge)
		return nil
	}

	log.Printf("[INFO] Sweeping %d of %d zones (name prefix %q, older than %s)", len(stale), len(zones), sweepZonePrefix, sweepZoneMinAge)

	var (
		mu                   sync.Mutex
		errs                 []error
		deleted, alreadyGone int
		wg                   sync.WaitGroup
		sem                  = make(chan struct{}, sweepConcurrency)
	)
	for _, zone := range stale {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			gone, err := deleteSweptZone(ctx, c, zone)
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

	log.Printf("[INFO] Deleted %d zones, %d already gone, %d failed", deleted, alreadyGone, len(errs))
	return errors.Join(errs...)
}

// deleteSweptZone deletes a zone. It reports alreadyGone when the zone no
// longer existed, which is the outcome the sweeper wants either way.
func deleteSweptZone(ctx context.Context, c *client.ClientWithResponses, zone client.Zone) (alreadyGone bool, err error) {
	resp, err := withSweepRetry(ctx, func() (*client.DeleteZoneResponse, error) {
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
