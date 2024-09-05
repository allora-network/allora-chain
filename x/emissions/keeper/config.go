package keeper 

type Config struct {
	// MaxMetadataLen defines the amount of characters that can be used for metadata
	MaxMetadataLen uint64
}

func DefaultConfig() Config {
	return Config{
		MaxMetadataLen:                       255,
	}
}
