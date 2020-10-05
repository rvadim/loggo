package storage

import (
	"fmt"
	"log"
	"time"

	"github.com/boltdb/bolt"
)

// RegistryFile abstraction over file which store log position of each log file
type RegistryFile struct {
	path       string
	db         *bolt.DB
	timeout    time.Duration
	bucketName []byte
}

// NewRegistryFile create new position file
func NewRegistryFile(path string, timeout time.Duration) (*RegistryFile, error) {
	r := RegistryFile{path: path}
	r.bucketName = []byte("logfiles")
	r.timeout = timeout
	var err error
	r.db, err = bolt.Open(path, 0600, &bolt.Options{Timeout: r.timeout * time.Second})
	if err != nil {
		log.Fatalln(err)
	}
	err = r.db.Update(func(tx *bolt.Tx) error {
		_, cerr := tx.CreateBucket(r.bucketName)
		if cerr != nil {
			if cerr.Error() == "bucket already exists" {
				return nil
			}
			return fmt.Errorf("create bucket: %s", cerr)
		}
		return nil
	})
	if err != nil {
		log.Fatalln(err)
	}
	return &r, nil
}

// GetAllKeys returns all keys from registry or empty array with error
func (r *RegistryFile) GetAllKeys() ([]string, error) {
	var output []string
	err := r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(r.bucketName)
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			output = append(output, string(k))
		}
		return nil
	})
	return output, err
}

// Delete delete row from registry by key
func (r *RegistryFile) Delete(key string) error {
	err := r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(r.bucketName)
		cerr := b.Delete([]byte(key))
		if cerr != nil {
			return cerr
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// Get return value by key from registry
func (r *RegistryFile) Get(key string) (string, error) {
	var output string
	err := r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(r.bucketName)
		v := b.Get([]byte(key))
		output = string(v)
		return nil
	})
	if err != nil {
		return "", err
	}
	return output, nil
}

// Set update value by key in registry
func (r *RegistryFile) Set(key string, value string) error {
	err := r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(r.bucketName)
		cerr := b.Put([]byte(key), []byte(value))
		if cerr != nil {
			return cerr
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// Close closes storage
func (r *RegistryFile) Close() error {
	err := r.db.Close()
	if err != nil {
		return err
	}
	return nil
}
