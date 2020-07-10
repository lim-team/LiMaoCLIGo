package client

// Options Options
type Options struct {
}

// NewOptions 创建默认配置
func NewOptions() *Options {
	return &Options{}
}

// Option 参数项
type Option func(*Options) error
