package adapter

import (
	"encoding/hex"
	"path"
	"strings"

	"github.com/hanfei1991/microcosm/pkg/errors"
)

var (
	MasterCampaignKey KeyAdapter = keyHexEncoderDecoder("/data-flow/master/leader/")
	// TODO: investigate whether we can merge MasterInfoKey and MasterMetaKey into one key
	MasterInfoKey      KeyAdapter = keyHexEncoderDecoder("/data-flow/master/info/")
	MasterMetaKey      KeyAdapter = keyHexEncoderDecoder("/data-flow/master/meta/")
	NodeInfoKeyAdapter KeyAdapter = keyHexEncoderDecoder("/data-flow/node/info/")
	JobKeyAdapter      KeyAdapter = keyHexEncoderDecoder("/data-flow/job/")
	TaskKeyAdapter     KeyAdapter = keyHexEncoderDecoder("/data-flow/task/")
	WorkerKeyAdapter   KeyAdapter = keyHexEncoderDecoder("/data-flow/worker/")

	ResourceKeyAdapter KeyAdapter = keyHexEncoderDecoder("/data-flow/resources/")

	// TODO: discuss the key prefix
	DMJobKeyAdapter KeyAdapter = keyHexEncoderDecoder("/data-flow/dm/job/")
)

type KeyAdapter interface {
	Encode(keys ...string) string
	Decode(key string) ([]string, error)
	Path() string
}

type keyHexEncoderDecoder string

func (s keyHexEncoderDecoder) Encode(keys ...string) string {
	hexKeys := []string{string(s)}
	for _, key := range keys {
		hexKeys = append(hexKeys, hex.EncodeToString([]byte(key)))
	}
	ret := path.Join(hexKeys...)
	//if len(keys) < keyAdapterKeysLen(s) {
	//	ret += "/"
	//}
	return ret
}

func (s keyHexEncoderDecoder) Decode(key string) ([]string, error) {
	if key[len(key)-1] == '/' {
		key = key[:len(key)-1]
	}
	v := strings.Split(strings.TrimPrefix(key, string(s)), "/")
	//if l := keyAdapterKeysLen(s); l != len(v) {
	//	return nil, terror.ErrDecodeEtcdKeyFail.Generate(fmt.Sprintf("decoder is %s, the key is %s", string(s), key))
	//}
	for i, k := range v {
		dec, err := hex.DecodeString(k)
		if err != nil {
			return nil, errors.Wrap(errors.ErrDecodeEtcdKeyFail, err, k)
		}
		v[i] = string(dec)
	}
	return v, nil
}

func (s keyHexEncoderDecoder) Path() string {
	return string(s)
}
