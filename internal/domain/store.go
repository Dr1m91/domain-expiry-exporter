package domain

// Store persists what the exporter knows about monitored domains.
type Store interface {
	Get(domain string) (Entry, bool, error)
	Set(entry Entry) error
	Delete(domain string) error
	List() ([]Entry, error)
}
