# Generation pipeline

The generation vertical slice is provider-independent and asynchronous:

`Studio → POST /generations → PostgreSQL → Redis/Asynq → worker → provider → Object Storage → asset → provenance`.

## Provider boundary

The domain depends only on `generation.Provider`. The YandexART implementation lives under `internal/providers/yandexart` and maps the provider's async operation and image response into domain DTOs.

YandexART uses the Foundation Models asynchronous image-generation API. The provider stores the provider operation ID and model version as provenance metadata.

## Idempotency

Every generation request requires an `Idempotency-Key`. The database enforces uniqueness per user. A repeated request returns the existing generation instead of enqueueing a duplicate job.

## Queue and retry

Redis is transport only. PostgreSQL remains the source of truth for generation state. Transient provider failures are returned to Asynq for retry; invalid requests are persisted as failed without retry.

## Assets

Successful provider images are validated before storage. The asset pipeline records:

- MIME type;
- byte size;
- dimensions;
- SHA-256 checksum;
- selected EXIF metadata;
- original, preview and thumbnail storage keys.

The checksum is unique per user for deduplication.

## Provenance

Generation creation and completion/failure are recorded in the hash-chained `provenance_events` table. The payload includes provider/model and reproducibility parameters without storing provider credentials.

## Configuration

Required for the real worker:

- `DATABASE_URL`
- `REDIS_URL`
- `S3_ENDPOINT`
- `S3_REGION`
- `S3_BUCKET`
- `S3_ACCESS_KEY`
- `S3_SECRET_KEY`
- `YANDEXART_API_KEY`
- `YANDEXART_FOLDER_ID`

Optional:

- `YANDEXART_ENDPOINT` (defaults to the Yandex Cloud Foundation Models endpoint)
- `YANDEXART_OPERATION_ENDPOINT` (defaults to the Yandex Cloud operation endpoint)
- `YANDEXART_MODEL` (defaults to `yandex-art/latest`)

CI and local tests use fakes/mocks; no paid YandexART request is required.
