// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// A simple example exposing fictional RPC latencies with different types of
// random distributions (uniform, normal, and exponential) as Prometheus
// metrics.

package main

import (
	"flag"
	"log"
	"math"
	"net/http"
	"time"

    "github.com/aptible/supercronic/cronexpr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		addr              = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
		oscillationPeriod = flag.Duration("oscillation-period", 5*time.Minute, "The duration of the rate oscillation period.")
		// cronExpr = flag.String("cron-expression", "*/5 * * * *", "Cron expression.")
		cronExpr = cronexpr.MustParse("0-5,10-15,20-25,30-35,40-45,50-55 * * * *")
	)

    flag.Func("cron-expression", "Cron expression", func(flagValue string) error {
        var err error
        cronExpr, err = cronexpr.Parse(flagValue)
        return err
    })

	flag.Parse()

	start := time.Now()

	var oscillationFunc = func() float64 {
		return math.Sin(2*math.Pi*float64(time.Since(start)) / float64(*oscillationPeriod))
	}

	var rpcDurations = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "sin",
			Help: "Oscillating sin function.",
		},
		oscillationFunc,
	)

	prometheus.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "epoch_seconds",
			Help: "Seconds since Unix epoch.",
		},
		func() float64 { return float64(time.Now().Unix()) },
	))
	// Register the summary and the histogram with Prometheus's default registry.
	prometheus.MustRegister(rpcDurations)

	var timeInterval = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "time_interval",
			Help: "",
		},
		func() float64 {
			now := time.Now()
			nextTime := cronExpr.Next(now)
			if nextTime.Sub(now) < 1*time.Minute {
				return 1
			}
			return 0
		},
	)
	prometheus.MustRegister(timeInterval)
	// Add Go module build info.
	prometheus.MustRegister(collectors.NewBuildInfoCollector())

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
