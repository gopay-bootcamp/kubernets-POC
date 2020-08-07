package config

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func GetInt64Ref(viper *viper.Viper, key string) *int64 {
	value := viper.GetInt64(key)
	return &value
}

func GetInt32Ref(viper *viper.Viper, key string) *int32 {
	value := viper.GetInt32(key)
	return &value
}

func GetMapFromJson(viper *viper.Viper, key string) map[string]string {
	var jsonStr = []byte(viper.GetString(key))
	var annotations map[string]string

	err := json.Unmarshal(jsonStr, &annotations)
	if err != nil {
		_ = fmt.Errorf("invalid Value for key %s, errors %v", key, err.Error())
	}

	return annotations
}

var once sync.Once
var config ProctorConfig

type ProctorConfig struct {
	viper 							*viper.Viper
	KubeConfig 						string
	KubeContext						string
	LogLevel 						string
	AppPort 						string
	DefaultNamespace 				string
	KubeWaitForResourcePollCount 	int
	KubeLogProcessWaitTime 			time.Duration
	KubeJobActiveDeadlineSeconds	*int64
	KubeJobRetries					*int32
	KubeServiceAccountName			string
	JobPodAnnotations				map[string]string
}

func load() ProctorConfig {
	fang := viper.New()
	fang.SetEnvPrefix("PROCTOR")
	fang.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	fang.AutomaticEnv()
	fang.SetConfigName("config")
	fang.AddConfigPath(".")

	value, available := os.LookupEnv("CONFIG_LOCATION")
	if available {
		fang.AddConfigPath(value)
	}

	_ = fang.ReadInConfig()

	proctorConfig := ProctorConfig{
		viper:	fang,
		KubeConfig: fang.GetString("kube.config"),
		KubeContext: fang.GetString("kube.context"),
		LogLevel: fang.GetString("log.level"),
		AppPort: fang.GetString("app.port"),
		DefaultNamespace: fang.GetString("default.namespace"),
		KubeWaitForResourcePollCount: fang.GetInt("kube.wait.for.resource.poll.count"),
		KubeJobActiveDeadlineSeconds: GetInt64Ref(fang, "kube.job.active.deadline.seconds"),
		KubeJobRetries: GetInt32Ref(fang, "kube.job.retries"),
		KubeLogProcessWaitTime: time.Duration(fang.GetInt("kube.log.process.wait.time")),
		KubeServiceAccountName: fang.GetString("kube.service.account.name"),
		JobPodAnnotations: GetMapFromJson(fang, "job.pod.annotations"),
	}

	return proctorConfig
}

/*
 *	Instead of using AtomBool, can we use channel and have a separate
 * 	goroutine to reset config?
 */

type AtomBool struct{ flag int32 }

func (b *AtomBool) Set(value bool) {
	var i int32 = 0
	if value {
		i = 1
	}
	atomic.StoreInt32(&(b.flag), i)
}

func (b *AtomBool) Get() bool {
	if atomic.LoadInt32(&(b.flag)) != 0 {
		return true
	}
	return false
}

var reset = new(AtomBool)

func init() {
	reset.Set(false)
}

func Reset() {
	reset.Set(true)
}

func Config() ProctorConfig {
	once.Do(func() {
		config = load()
	})

	if reset.Get() {
		config = load()
		reset.Set(false)
	}

	return config
}