package assets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/image/draw"

	"github.com/oleg3190/Web-studio-img/backend/internal/storage"
)

const MaxAssetSize int64 = 10 << 20

type Asset struct {
	ID           uuid.UUID       `json:"id"`
	ProjectID    uuid.UUID       `json:"project_id"`
	GenerationID uuid.UUID       `json:"generation_id"`
	UserID       uuid.UUID       `json:"user_id"`
	StorageKey   string          `json:"storage_key"`
	PreviewKey   *string         `json:"preview_key,omitempty"`
	ThumbnailKey *string         `json:"thumbnail_key,omitempty"`
	MIMEType     string          `json:"mime_type"`
	Size         int64           `json:"size"`
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	Checksum     string          `json:"checksum"`
	EXIF         map[string]any  `json:"exif,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("asset store requires database")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, userID, projectID, generationID uuid.UUID, storageKey, previewKey, thumbnailKey, mime string, size int64, width, height int, checksum string, exifData map[string]any) (Asset, bool, error) {
	raw, err := json.Marshal(exifData)
	if err != nil {
		return Asset{}, false, err
	}
	var item Asset
	var storedRaw []byte
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO assets
			(user_id, project_id, generation_id, storage_key, preview_key, thumbnail_key, mime_type, size, width, height, checksum, exif)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (user_id, checksum) DO NOTHING
		RETURNING id, project_id, generation_id, user_id, storage_key, preview_key, thumbnail_key, mime_type, size, width, height, checksum, exif, created_at
	`, userID, projectID, generationID, storageKey, previewKey, thumbnailKey, mime, size, width, height, checksum, raw).Scan(
		&item.ID, &item.ProjectID, &item.GenerationID, &item.UserID, &item.StorageKey, &item.PreviewKey, &item.ThumbnailKey,
		&item.MIMEType, &item.Size, &item.Width, &item.Height, &item.Checksum, &storedRaw, &item.CreatedAt,
	)
	if err == nil {
		_ = json.Unmarshal(storedRaw, &item.EXIF)
		return item, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Asset{}, false, fmt.Errorf("create asset: %w", err)
	}
	err = s.db.QueryRowContext(ctx, `
		SELECT id, project_id, generation_id, user_id, storage_key, preview_key, thumbnail_key, mime_type, size, width, height, checksum, exif, created_at
		FROM assets WHERE user_id=$1 AND checksum=$2
	`, userID, checksum).Scan(
		&item.ID, &item.ProjectID, &item.GenerationID, &item.UserID, &item.StorageKey, &item.PreviewKey, &item.ThumbnailKey,
		&item.MIMEType, &item.Size, &item.Width, &item.Height, &item.Checksum, &storedRaw, &item.CreatedAt,
	)
	if err != nil {
		return Asset{}, false, fmt.Errorf("load duplicate asset: %w", err)
	}
	_ = json.Unmarshal(storedRaw, &item.EXIF)
	return item, false, nil
}

type Processor struct {
	Storage storage.StorageProvider
	Store   *Store
}

func (p *Processor) Process(ctx context.Context, userID, projectID, generationID uuid.UUID, imageData []byte, mime string) (Asset, error) {
	if len(imageData) == 0 || int64(len(imageData)) > MaxAssetSize {
		return Asset{}, fmt.Errorf("asset exceeds %d bytes", MaxAssetSize)
	}
	detected := httpDetectContentType(imageData)
	if detected != "image/jpeg" && detected != "image/png" {
		return Asset{}, fmt.Errorf("unsupported image MIME type %q", detected)
	}
	mime = detected
	cfg, format, err := image.DecodeConfig(bytes.NewReader(imageData))
	if err != nil {
		return Asset{}, fmt.Errorf("decode image: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 8192 || cfg.Height > 8192 {
		return Asset{}, fmt.Errorf("invalid image dimensions")
	}
	if format != "jpeg" && format != "png" {
		return Asset{}, fmt.Errorf("unsupported image format %q", format)
	}
	sum := sha256.Sum256(imageData)
	checksum := hex.EncodeToString(sum[:])
	base := fmt.Sprintf("projects/%s/generations/%s/%s", projectID, generationID, checksum)
	originalKey := base + "/original"
	previewKey := base + "/preview.jpg"
	thumbnailKey := base + "/thumbnail.jpg"
	exifData := readEXIF(imageData)

	if err := p.Storage.Put(ctx, originalKey, bytes.NewReader(imageData), int64(len(imageData)), storage.PutOptions{ContentType: mime, Metadata: map[string]string{"sha256": checksum}}); err != nil {
		return Asset{}, err
	}
	preview, err := makePreview(imageData, 1600)
	if err != nil {
		return Asset{}, err
	}
	thumbnail, err := makePreview(imageData, 320)
	if err != nil {
		return Asset{}, err
	}
	if err := p.Storage.Put(ctx, previewKey, bytes.NewReader(preview), int64(len(preview)), storage.PutOptions{ContentType: "image/jpeg"}); err != nil {
		return Asset{}, err
	}
	if err := p.Storage.Put(ctx, thumbnailKey, bytes.NewReader(thumbnail), int64(len(thumbnail)), storage.PutOptions{ContentType: "image/jpeg"}); err != nil {
		return Asset{}, err
	}
	item, created, err := p.Store.Create(ctx, userID, projectID, generationID, originalKey, previewKey, thumbnailKey, mime, int64(len(imageData)), cfg.Width, cfg.Height, checksum, exifData)
	if err != nil {
		return Asset{}, err
	}
	if !created {
		return item, nil
	}
	return item, nil
}

func makePreview(data []byte, maxWidth int) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width > maxWidth {
		height = height * maxWidth / width
		width = maxWidth
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 88}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func readEXIF(data []byte) map[string]any {
	result := map[string]any{}
	x, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return result
	}
	for _, tag := range []exif.FieldName{exif.Make, exif.Model, exif.DateTimeOriginal, exif.Software, exif.Orientation} {
		if value, err := x.Get(tag); err == nil {
			result[string(tag)] = strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return result
}

func httpDetectContentType(data []byte) string {
	if len(data) > 512 {
		data = data[:512]
	}
	return strings.TrimSpace(httpContentType(data))
}

func httpContentType(data []byte) string {
	if len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
		return "image/png"
	}
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return "image/jpeg"
	}
	return "application/octet-stream"
}

