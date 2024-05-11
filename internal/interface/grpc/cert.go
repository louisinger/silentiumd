package grpcservice

import (
	"context"

	"cloud.google.com/go/datastore"
	"golang.org/x/crypto/acme/autocert"
)

type (
	LetsencryptCert struct {
		Data []byte `datastore:"data,noindex"`
	}

	datastoreCertCache struct {
		client *datastore.Client
	}
)

func newDatastoreCertCache(client *datastore.Client) *datastoreCertCache {
	return &datastoreCertCache{
		client: client,
	}
}

func (d *datastoreCertCache) Get(ctx context.Context, key string) ([]byte, error) {
	var cert LetsencryptCert
	k := datastore.NameKey("letsencrypt", key, nil)
	if err := d.client.Get(ctx, k, &cert); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, autocert.ErrCacheMiss
		}
		return nil, err
	}
	return cert.Data, nil
}

// Put stores the data in the cache under the specified key.
// Underlying implementations may use any data storage format,
// as long as the reverse operation, Get, results in the original data.
func (d *datastoreCertCache) Put(ctx context.Context, key string, data []byte) error {
	k := datastore.NameKey("letsencrypt", key, nil)
	cert := LetsencryptCert{
		Data: data,
	}
	if _, err := d.client.Put(ctx, k, &cert); err != nil {
		return err
	}
	return nil
}

// Delete removes a certificate data from the cache under the specified key.
// If there's no such key in the cache, Delete returns nil.
func (d *datastoreCertCache) Delete(ctx context.Context, key string) error {
	k := datastore.NameKey("letsencrypt", key, nil)
	if err := d.client.Delete(ctx, k); err != nil {
		return err
	}
	return nil
}
