package dependency

import (
	"fmt"
	"log"
	"net/url"
	"regexp"

	"github.com/pkg/errors"
)

var (
	// Ensure implements
	_ Dependency = (*KVGetQuery)(nil)

	// KVGetQueryRe is the regular expression to use.
	KVGetQueryRe = regexp.MustCompile(`\A` + keyRe + dcRe + `\z`)
)

// KVGetQuery queries the KV store for a single key.
type KVGetQuery struct {
	stopCh chan struct{}

	dc  string
	key string
}

// NewKVGetQuery parses a string into a dependency.
func NewKVGetQuery(s string) (*KVGetQuery, error) {
	if s != "" && !KVGetQueryRe.MatchString(s) {
		return nil, fmt.Errorf("kv.get: invalid format: %q", s)
	}

	m := regexpMatch(KVGetQueryRe, s)
	return &KVGetQuery{
		dc:     m["dc"],
		key:    m["key"],
		stopCh: make(chan struct{}, 1),
	}, nil
}

// Fetch queries the Consul API defined by the given client.
func (d *KVGetQuery) Fetch(clients *ClientSet, opts *QueryOptions) (interface{}, *ResponseMetadata, error) {
	select {
	case <-d.stopCh:
		return nil, nil, ErrStopped
	default:
	}

	opts = opts.Merge(&QueryOptions{
		Datacenter: d.dc,
	})

	log.Printf("[TRACE] %s: GET %s", d, &url.URL{
		Path:     "/v1/kv/" + d.key,
		RawQuery: opts.String(),
	})

	pair, qm, err := clients.Consul().KV().Get(d.key, opts.ToConsulOpts())
	if err != nil {
		return nil, nil, errors.Wrap(err, d.String())
	}

	rm := &ResponseMetadata{
		LastIndex:   qm.LastIndex,
		LastContact: qm.LastContact,
	}

	if pair == nil {
		log.Printf("[TRACE] %s: returned nil", d)
		return nil, rm, nil
	}

	value := string(pair.Value)
	log.Printf("[TRACE] %s: returned %q", d, value)
	return value, rm, nil
}

// CanShare returns a boolean if this dependency is shareable.
func (d *KVGetQuery) CanShare() bool {
	return true
}

// String returns the human-friendly version of this dependency.
func (d *KVGetQuery) String() string {
	key := d.key
	if d.dc != "" {
		key = key + "@" + d.dc
	}
	return fmt.Sprintf("kv.get(%s)", key)
}

// HashCode returns a unique identifier.
func (d *KVGetQuery) HashCode() string {
	return d.String()
}

// Display prints the human-friendly output.
func (d *KVGetQuery) Display() string {
	return d.String()
}

// Stop halts the dependency's fetch function.
func (d *KVGetQuery) Stop() {
	close(d.stopCh)
}