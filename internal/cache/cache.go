package cache

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ideamans/crux-cli/internal/crux"
)

const latestMonthTTL = 24 * time.Hour

// LatestMonth holds the most recent month available in the CrUX dataset.
type LatestMonth struct {
	Yyyymm    string    `json:"yyyymm"`
	CheckedAt time.Time `json:"checked_at"`
}

// Cache manages local cache files under a given directory.
type Cache struct {
	dir string
}

func New(dir string) *Cache {
	return &Cache{dir: dir}
}

func (c *Cache) Dir() string {
	return c.dir
}

// originHash returns a 16-char hex prefix of the SHA256 of origin.
func (c *Cache) originHash(origin string) string {
	h := sha256.Sum256([]byte(origin))
	return fmt.Sprintf("%x", h[:8])
}

func (c *Cache) latestMonthPath() string {
	return filepath.Join(c.dir, "latest_month.json")
}

// deviceCachePath returns the gzipped JSON cache path for a given origin and period.
// The filename encodes (monthFrom_monthTo), so a change in either month produces a cache miss.
func (c *Cache) deviceCachePath(origin, monthFrom, monthTo string) string {
	hash := c.originHash(origin)
	return filepath.Join(c.dir, "device", hash, fmt.Sprintf("%s_%s.json.gz", monthFrom, monthTo))
}

// GetLatestMonth reads the cached latest month. Returns nil if not found.
func (c *Cache) GetLatestMonth() (*LatestMonth, error) {
	data, err := os.ReadFile(c.latestMonthPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var lm LatestMonth
	if err := json.Unmarshal(data, &lm); err != nil {
		return nil, err
	}
	return &lm, nil
}

// SaveLatestMonth writes the latest month to disk.
func (c *Cache) SaveLatestMonth(lm *LatestMonth) error {
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lm, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.latestMonthPath(), data, 0644)
}

// IsStale returns true when the latest_month cache is older than 24 hours or missing.
func (c *Cache) IsStale(lm *LatestMonth) bool {
	return lm == nil || time.Since(lm.CheckedAt) > latestMonthTTL
}

// GetDevice reads the cached rows for an origin+period. Returns nil if not cached.
func (c *Cache) GetDevice(origin, monthFrom, monthTo string) ([]crux.DeviceRow, error) {
	path := c.deviceCachePath(origin, monthFrom, monthTo)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	var rows []crux.DeviceRow
	if err := json.NewDecoder(gz).Decode(&rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// SaveDevice writes rows for an origin+period as a gzip-compressed JSON file.
func (c *Cache) SaveDevice(origin, monthFrom, monthTo string, rows []crux.DeviceRow) error {
	path := c.deviceCachePath(origin, monthFrom, monthTo)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	if err := json.NewEncoder(gz).Encode(rows); err != nil {
		gz.Close()
		return err
	}
	return gz.Close()
}

// ListFiles returns relative paths of all device cache files.
func (c *Cache) ListFiles() ([]string, error) {
	deviceDir := filepath.Join(c.dir, "device")
	var files []string
	err := filepath.Walk(deviceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(c.dir, path)
			files = append(files, rel)
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	return files, err
}

// ClearAll removes all device cache files.
func (c *Cache) ClearAll() error {
	return os.RemoveAll(filepath.Join(c.dir, "device"))
}

// ClearOrigin removes cache files for a specific origin.
func (c *Cache) ClearOrigin(origin string) error {
	hash := c.originHash(origin)
	return os.RemoveAll(filepath.Join(c.dir, "device", hash))
}
