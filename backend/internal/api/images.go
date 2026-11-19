package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"backend/internal/forum"
)

const imageCacheLimit = 80 << 20

type cachedImage struct {
	data        []byte
	contentType string
}

type imageCache struct {
	mu    sync.Mutex
	items map[string]cachedImage
	order []string
	size  int
}

func newImageCache() *imageCache {
	return &imageCache{items: map[string]cachedImage{}}
}

func (c *imageCache) get(key string) (cachedImage, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[key]
	return item, ok
}

func (c *imageCache) put(key string, item cachedImage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.items[key]; ok {
		return
	}

	c.items[key] = item
	c.order = append(c.order, key)
	c.size += len(item.data)

	for c.size > imageCacheLimit && len(c.order) > 1 {
		oldest := c.order[0]
		c.order = c.order[1:]
		c.size -= len(c.items[oldest].data)
		delete(c.items, oldest)
	}
}

func (h *Handler) image(w http.ResponseWriter, r *http.Request) {
	imageURL := r.URL.Query().Get("url")

	if imageURL == "" || !h.store.KnownImage(imageURL) {
		writeError(w, http.StatusNotFound, "Изображение не найдено")
		return
	}

	item, ok := h.images.get(imageURL)
	if !ok {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		data, contentType, err := h.fetcher.FetchImage(ctx, imageURL)
		if err != nil {
			log.Printf("Не удалось загрузить изображение %s: %v", imageURL, err)

			status := http.StatusBadGateway
			if errors.Is(err, forum.ErrNotImage) {
				status = http.StatusUnsupportedMediaType
			}
			writeError(w, status, "Не удалось загрузить изображение")
			return
		}

		item = cachedImage{data: data, contentType: contentType}
		h.images.put(imageURL, item)
	}

	w.Header().Set("Content-Type", item.contentType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(item.data)
}
