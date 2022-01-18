package global

import "github.com/AdmiralBulldogTv/VodEdge/src/instance"

type Instances struct {
	Redis      instance.Redis
	Mongo      instance.Mongo
	Prometheus instance.Prometheus
}
