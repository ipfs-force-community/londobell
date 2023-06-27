package common

type Options struct {
	BatchInsertLimit int
}

func NewOptions(batchInsertLimit int) Options {
	return Options{BatchInsertLimit: batchInsertLimit}
}
