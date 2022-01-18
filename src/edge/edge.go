package edge

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"sort"
	"time"

	"github.com/AdmiralBulldogTv/VodEdge/src/global"
	"github.com/AdmiralBulldogTv/VodEdge/src/structures"
	"github.com/AdmiralBulldogTv/VodEdge/src/svc/mongo"
	"github.com/AdmiralBulldogTv/VodEdge/src/utils"
	"github.com/fasthttp/router"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func New(gCtx global.Context) <-chan struct{} {
	done := make(chan struct{})

	// cdn.admiralbulldog.live/vods/<vod-id>/master.m3u8
	// cdn.admiralbulldog.live/vods/<vod-id>/<variant>/playlist.m3u8
	// cdn.admiralbulldog.live/vods/<vod-id>/<variant>/<seg>.ts

	r := router.New()
	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
	}

	r.GET("/{vod}/master.m3u8", func(ctx *fasthttp.RequestCtx) {
		vID, err := primitive.ObjectIDFromHex(ctx.UserValue("vod").(string))
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}

		res := gCtx.Inst().Mongo.Collection(mongo.CollectionNameVods).FindOne(ctx, bson.M{
			"_id": vID,
		})
		vod := structures.Vod{}
		err = res.Err()
		if err == nil {
			err = res.Decode(&vod)
		}
		if err != nil {
			if err == mongo.ErrNoDocuments {
				ctx.SetStatusCode(fasthttp.StatusNotFound)
				return
			}

			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			return
		}

		if len(vod.Variants) == 0 {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}

		readyVariants := []structures.VodVariant{}
		for _, v := range vod.Variants {
			if v.Ready {
				readyVariants = append(readyVariants, v)
			}
		}

		if len(readyVariants) == 0 {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}

		sort.Slice(readyVariants, func(i, j int) bool {
			return readyVariants[i].Bitrate < readyVariants[j].Bitrate
		})

		b := bytes.NewBuffer(nil)
		_, _ = b.WriteString("#EXTM3U\n")

		for _, v := range readyVariants {
			_, _ = b.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,AVERAGE-BANDWIDTH=%d,RESOLUTION=%dx%d,FRAME-RATE=%d,CODECS=\"avc1.42e00a,mp4a.40.2\"\n", v.Bitrate, v.Bitrate, v.Width, v.Height, v.FPS))
			_, _ = b.WriteString(fmt.Sprintf("%s/playlist.m3u8\n", v.Name))
		}

		ctx.SetBody(b.Bytes())
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.Response.Header.Set("Content-Type", "application/x-mpegURL")
		ctx.Response.Header.Set("Cache-Control", "public, max-age=120, s-max-age=120")
	})

	r.GET("/{vod}/{variant}/playlist.m3u8", func(ctx *fasthttp.RequestCtx) {
		vID, err := primitive.ObjectIDFromHex(ctx.UserValue("vod").(string))
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}

		f, err := os.OpenFile(path.Join(gCtx.Config().Edge.ProcessedVodsPath, vID.Hex(), path.Clean(ctx.UserValue("variant").(string)), "playlist.m3u8"), os.O_RDONLY, 0644)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}

		ctx.Response.Header.Set("Cache-Control", "public, max-age=604800, s-max-age=604800, immutable")
		ctx.Response.Header.Set("Content-Type", "application/x-mpegURL")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyStream(f, -1)
	})

	r.GET("/{vod}/{variant}/{seg}.ts", func(ctx *fasthttp.RequestCtx) {
		vID, err := primitive.ObjectIDFromHex(ctx.UserValue("vod").(string))
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}

		f, err := os.OpenFile(path.Join(gCtx.Config().Edge.ProcessedVodsPath, vID.Hex(), path.Clean(ctx.UserValue("variant").(string)), path.Clean(ctx.UserValue("seg").(string)+".ts")), os.O_RDONLY, 0644)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			return
		}

		ctx.Response.Header.Set("Cache-Control", "public, max-age=604800, s-max-age=604800, immutable")
		ctx.Response.Header.Set("Content-Type", "video/MP2T")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyStream(f, -1)
	})

	server := fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			start := time.Now()
			defer func() {
				gCtx.Inst().Prometheus.ResponseTimeMilliseconds().Observe(float64(time.Since(start)/time.Microsecond) / 1000)
				l := logrus.WithFields(logrus.Fields{
					"status":     ctx.Response.StatusCode(),
					"duration":   time.Since(start) / time.Millisecond,
					"entrypoint": "edge",
					"path":       utils.B2S(ctx.Path()),
				})
				if err := recover(); err != nil {
					l.Error("panic in handler: ", err)
				} else {
					l.Info("")
				}
				switch ctx.Response.StatusCode() {
				case fasthttp.StatusNotFound:
					ctx.Response.Header.Set("Cache-Control", "public, max-age=120, s-max-age=120")
				}
			}()
			ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
			ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET,OPTIONS")
			if ctx.IsOptions() {
				ctx.SetStatusCode(fasthttp.StatusNoContent)
				return
			}
			r.Handler(ctx)
		},
		Name:            "Troy",
		ReadTimeout:     time.Second * 10,
		WriteTimeout:    time.Second * 10,
		IdleTimeout:     time.Second * 10,
		GetOnly:         true,
		CloseOnShutdown: true,
	}

	go func() {
		if err := server.ListenAndServe(gCtx.Config().Edge.Bind); err != nil {
			logrus.Fatal("failed to start edge server: ", err)
		}
		close(done)
	}()

	go func() {
		<-gCtx.Done()
		_ = server.Shutdown()
	}()

	return done
}
