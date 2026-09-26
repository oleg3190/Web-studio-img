// Package storage defines the object-storage port and its S3-compatible adapter.
//
// The adapter targets Yandex Object Storage in production and MinIO for local
// development. Presigned URLs keep object transfer off the API data path.
package storage
