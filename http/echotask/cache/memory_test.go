package cache

import (
	"reflect"
	"testing"
	"time"
)

func TestMemoryGet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		key   string
		setup func(cache *Memory)
		want  []byte
		want1 bool
	}{
		{
			name: "Key exists in cache",
			setup: func(cache *Memory) {
				cache.Set("key1", []byte("value1"))
			},
			key:   "key1",
			want:  []byte("value1"),
			want1: true,
		},
		{
			name:  "Key does not exist in cache",
			setup: func(cache *Memory) {},
			key:   "key1",
			want:  nil,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewMemory(100, time.Second)

			tt.setup(c)

			got, got1 := c.Get(tt.key)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Get() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestMemorySet(t *testing.T) {
	t.Parallel()
	type args struct {
		key     string
		content []byte
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Set key with value",
			args: args{
				key:     "key1",
				content: []byte("value1"),
			},
		},
		{
			name: "Set key with empty string",
			args: args{
				key:     "key2",
				content: []byte(""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewMemory(100, time.Second)
			c.Set(tt.args.key, tt.args.content)

			got, got1 := c.Get(tt.args.key)
			if !reflect.DeepEqual(got, tt.args.content) {
				t.Errorf("Set() Get() got = %v, want %v", got, tt.args.content)
			}
			if !got1 {
				t.Errorf("Set() Get() key %v does not exist, but it should", tt.args.key)
			}
		})
	}
}

func TestNewMemory(t *testing.T) {
	t.Parallel()
	cache := NewMemory(100, time.Minute)

	if cache == nil {
		t.Errorf("NewMemory() returned nil, expected valid cache instance")
		return
	}

	if cache.cache.Len() != 0 {
		t.Errorf("NewMemory() initialized cache data with length %d, expected 0", cache.cache.Len())
	}
}
