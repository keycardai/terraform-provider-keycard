// Command sweep deletes what acceptance tests left behind in the
// organization the KEYCARD_* credentials belong to: test zones, and test
// resources, applications, providers and groups in the organization zone.
//
// A test leaves objects behind when it never reaches its own destroy: a go
// test timeout panics the whole binary, a cancelled CI job kills it, and a
// failed child delete stops Terraform. A create that fails on the client
// after the server created the object also leaves it behind, because the
// object never reaches the Terraform state.
//
// Usage:
//
//	go run ./cmd/sweep -dry-run   # list what would be deleted
//	go run ./cmd/sweep            # delete it
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

	"github.com/oapi-codegen/nullable"
	"golang.org/x/oauth2"

	"github.com/keycardai/terraform-provider-keycard/internal/client"
)

const (
	// namePrefix matches the names acceptance tests give their objects.
	// acctest.RandomWithPrefix("tftest...") always writes "<prefix>-<number>",
	// so every test object name starts with "tftest-".
	namePrefix = "tftest-"

	// minAge keeps the sweeper away from objects a running test still owns.
	// It is well above the acceptance job timeout, so any older object
	// belongs to a run that has already ended.
	minAge = time.Hour

	// concurrency bounds parallel deletes. Each zone delete deregisters the
	// zone from Vault and the router, so a large backlog is slow serially.
	concurrency = 5

	// requestTimeout replaces the provider client's 5s timeout. A zone delete
	// calls Vault and the router, which can take longer under load.
	requestTimeout = 30 * time.Second

	// attempts bounds retries of a list page or a delete after a transport
	// error, a 429 or a 5xx.
	attempts = 4

	// pageSize is the largest page the list APIs return.
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
	dryRun := fs.Bool("dry-run", false, "list what would be deleted, without deleting it")
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

// target is one object the sweeper may delete.
type target struct {
	id        string
	name      string
	createdAt time.Time
}

// reply is the status and body of an API response.
type reply struct {
	status int
	body   []byte
}

func (r reply) StatusCode() int { return r.status }

// listPage is one page of a list call. ok is false when the API did not
// return a list.
type listPage struct {
	reply
	ok      bool
	targets []target
	next    nullable.Nullable[string]
}

// collection is one kind of object to sweep, in one place.
type collection struct {
	kind   string
	list   func(ctx context.Context, after *string) (listPage, error)
	delete func(ctx context.Context, id string) (reply, error)
}

// sweep deletes what acceptance tests left behind, or only lists it when
// dryRun is set. Most tests create their objects in the organization zone
// instead of a zone of their own, so the sweeper deletes test objects there
// as well as test zones. Collections are swept in dependency order:
// resources, which reference applications and providers, go first.
func sweep(ctx context.Context, c *client.ClientWithResponses, dryRun bool) error {
	var errs []error
	cols := []collection{zoneCollection(c)}
	if orgZoneID, err := organizationZoneID(ctx, c); err != nil {
		errs = append(errs, err)
	} else {
		cols = append(orgZoneCollections(c, orgZoneID), cols...)
	}

	for _, col := range cols {
		if err := sweepCollection(ctx, col, dryRun); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// orgZoneCollections returns what to sweep in the organization zone, in
// dependency order.
func orgZoneCollections(c *client.ClientWithResponses, orgZoneID string) []collection {
	limit := pageSize
	return []collection{
		{
			kind: "resource",
			list: func(ctx context.Context, after *string) (listPage, error) {
				resp, err := c.ListResourcesWithResponse(ctx, orgZoneID, &client.ListResourcesParams{After: after, Limit: &limit})
				if err != nil {
					return listPage{}, err
				}
				page := listPage{reply: reply{resp.StatusCode(), resp.Body}}
				if resp.JSON200 != nil {
					page.ok = true
					page.next = resp.JSON200.Pagination.AfterCursor
					for _, r := range resp.JSON200.Items {
						page.targets = append(page.targets, target{r.Id, r.Name, r.CreatedAt})
					}
				}
				return page, nil
			},
			delete: func(ctx context.Context, id string) (reply, error) {
				resp, err := c.DeleteResourceWithResponse(ctx, orgZoneID, id)
				if err != nil {
					return reply{}, err
				}
				return reply{resp.StatusCode(), resp.Body}, nil
			},
		},
		{
			kind: "application",
			list: func(ctx context.Context, after *string) (listPage, error) {
				resp, err := c.ListApplicationsWithResponse(ctx, orgZoneID, &client.ListApplicationsParams{After: after, Limit: &limit})
				if err != nil {
					return listPage{}, err
				}
				page := listPage{reply: reply{resp.StatusCode(), resp.Body}}
				if resp.JSON200 != nil {
					page.ok = true
					page.next = resp.JSON200.Pagination.AfterCursor
					for _, a := range resp.JSON200.Items {
						page.targets = append(page.targets, target{a.Id, a.Name, a.CreatedAt})
					}
				}
				return page, nil
			},
			delete: func(ctx context.Context, id string) (reply, error) {
				resp, err := c.DeleteApplicationWithResponse(ctx, orgZoneID, id)
				if err != nil {
					return reply{}, err
				}
				return reply{resp.StatusCode(), resp.Body}, nil
			},
		},
		{
			kind: "provider",
			list: func(ctx context.Context, after *string) (listPage, error) {
				resp, err := c.ListProvidersWithResponse(ctx, orgZoneID, &client.ListProvidersParams{After: after, Limit: &limit})
				if err != nil {
					return listPage{}, err
				}
				page := listPage{reply: reply{resp.StatusCode(), resp.Body}}
				if resp.JSON200 != nil {
					page.ok = true
					page.next = resp.JSON200.Pagination.AfterCursor
					for _, p := range resp.JSON200.Items {
						page.targets = append(page.targets, target{p.Id, p.Name, p.CreatedAt})
					}
				}
				return page, nil
			},
			delete: func(ctx context.Context, id string) (reply, error) {
				resp, err := c.DeleteProviderWithResponse(ctx, orgZoneID, id)
				if err != nil {
					return reply{}, err
				}
				return reply{resp.StatusCode(), resp.Body}, nil
			},
		},
		{
			kind: "group",
			list: func(ctx context.Context, after *string) (listPage, error) {
				resp, err := c.ListGroupsWithResponse(ctx, orgZoneID, &client.ListGroupsParams{After: after, Limit: &limit})
				if err != nil {
					return listPage{}, err
				}
				page := listPage{reply: reply{resp.StatusCode(), resp.Body}}
				if resp.JSON200 != nil {
					page.ok = true
					page.next = resp.JSON200.Pagination.AfterCursor
					for _, g := range resp.JSON200.Items {
						page.targets = append(page.targets, target{g.Id, g.Name, g.CreatedAt})
					}
				}
				return page, nil
			},
			delete: func(ctx context.Context, id string) (reply, error) {
				resp, err := c.DeleteGroupWithResponse(ctx, orgZoneID, id)
				if err != nil {
					return reply{}, err
				}
				return reply{resp.StatusCode(), resp.Body}, nil
			},
		},
	}
}

// zoneCollection sweeps test zones. It does not need the organization zone,
// so it runs even when the organization lookup fails.
func zoneCollection(c *client.ClientWithResponses) collection {
	limit := pageSize
	return collection{
		kind: "zone",
		list: func(ctx context.Context, after *string) (listPage, error) {
			resp, err := c.ListZonesWithResponse(ctx, &client.ListZonesParams{After: after, Limit: &limit})
			if err != nil {
				return listPage{}, err
			}
			page := listPage{reply: reply{resp.StatusCode(), resp.Body}}
			if resp.JSON200 != nil {
				page.ok = true
				page.next = resp.JSON200.Pagination.AfterCursor
				for _, z := range resp.JSON200.Items {
					page.targets = append(page.targets, target{z.Id, z.Name, z.CreatedAt})
				}
			}
			return page, nil
		},
		delete: func(ctx context.Context, id string) (reply, error) {
			resp, err := c.DeleteZoneWithResponse(ctx, id)
			if err != nil {
				return reply{}, err
			}
			return reply{resp.StatusCode(), resp.Body}, nil
		},
	}
}

// organizationZoneID returns the built-in zone of the organization the
// credentials belong to.
func organizationZoneID(ctx context.Context, c *client.ClientWithResponses) (string, error) {
	resp, err := withRetry(ctx, func() (*client.ListOrganizationsResponse, error) {
		return c.ListOrganizationsWithResponse(ctx, &client.ListOrganizationsParams{})
	})
	if err != nil {
		return "", fmt.Errorf("listing organizations: %w", err)
	}
	if resp.JSON200 == nil {
		return "", fmt.Errorf("listing organizations, got status %d: %s", resp.StatusCode(), string(resp.Body))
	}
	if len(resp.JSON200.Items) != 1 {
		return "", fmt.Errorf("listing organizations: got %d, want 1", len(resp.JSON200.Items))
	}
	zoneID := resp.JSON200.Items[0].ZoneId
	if zoneID == nil || *zoneID == "" {
		return "", errors.New("the organization has no built-in zone")
	}
	return *zoneID, nil
}

// sweepCollection deletes the leftover test objects of one collection.
func sweepCollection(ctx context.Context, col collection, dryRun bool) error {
	all, err := listAll(ctx, col)
	if err != nil {
		return err
	}

	now := time.Now()
	var stale []target
	for _, t := range all {
		if sweepable(t, now) {
			stale = append(stale, t)
		}
	}

	if dryRun {
		for _, t := range stale {
			log.Printf("Would delete %s %s (%s), created %s", col.kind, t.id, t.name, t.createdAt.Format(time.RFC3339))
		}
		log.Printf("Dry run: would delete %d of %d %ss (name prefix %q, older than %s); nothing was deleted", len(stale), len(all), col.kind, namePrefix, minAge)
		return nil
	}

	log.Printf("Deleting %d of %d %ss (name prefix %q, older than %s)", len(stale), len(all), col.kind, namePrefix, minAge)

	var (
		mu                   sync.Mutex
		errs                 []error
		deleted, alreadyGone int
		wg                   sync.WaitGroup
		sem                  = make(chan struct{}, concurrency)
	)
	for _, t := range stale {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			gone, err := deleteTarget(ctx, col, t)
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

	log.Printf("Deleted %d %ss, %d already gone, %d failed", deleted, col.kind, alreadyGone, len(errs))
	return errors.Join(errs...)
}

// listAll returns every object in a collection. It lists everything before
// anything is deleted, so deletes can't shift the cursor.
func listAll(ctx context.Context, col collection) ([]target, error) {
	var all []target
	var after *string
	for range maxPages {
		page, err := withRetry(ctx, func() (listPage, error) {
			return col.list(ctx, after)
		})
		if err != nil {
			return nil, fmt.Errorf("listing %ss: %w", col.kind, err)
		}
		if !page.ok {
			return nil, fmt.Errorf("listing %ss, got status %d: %s", col.kind, page.status, string(page.body))
		}
		all = append(all, page.targets...)

		cursor := page.next
		if !cursor.IsSpecified() || cursor.IsNull() || cursor.MustGet() == "" {
			return all, nil
		}
		next := cursor.MustGet()
		after = &next
	}
	return nil, fmt.Errorf("listing %ss: still more pages after %d; stopping", col.kind, maxPages)
}

// sweepable reports whether an object was left behind by an acceptance test
// that has finished.
func sweepable(t target, now time.Time) bool {
	return strings.HasPrefix(t.name, namePrefix) && now.Sub(t.createdAt) >= minAge
}

// deleteTarget deletes one object. It reports alreadyGone when the object no
// longer existed, which is the outcome the sweeper wants either way.
func deleteTarget(ctx context.Context, col collection, t target) (alreadyGone bool, err error) {
	resp, err := withRetry(ctx, func() (reply, error) {
		return col.delete(ctx, t.id)
	})
	if err != nil {
		return false, fmt.Errorf("deleting %s %s (%s): %w", col.kind, t.id, t.name, err)
	}
	switch resp.status {
	case http.StatusNoContent:
		return false, nil
	case http.StatusNotFound:
		return true, nil
	default:
		return false, fmt.Errorf("deleting %s %s (%s), got status %d: %s", col.kind, t.id, t.name, resp.status, string(resp.body))
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
