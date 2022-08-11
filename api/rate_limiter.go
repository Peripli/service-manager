package api

import (
	"context"
	"fmt"
	"github.com/Peripli/service-manager/api/filters"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/ulule/limiter"
	"github.com/ulule/limiter/drivers/middleware/stdlib"
	"github.com/ulule/limiter/drivers/store/memory"
	"github.com/ulule/limiter/drivers/store/redis"
	"net/http"
	"path"
	"strings"
)

func validateRateLimiterConfiguration(config string) error {
	_, err := parseRateLimiterConfiguration(config)
	if err != nil {
		return err
	}
	return nil
}

type RateLimiterConfiguration struct {
	rate       limiter.Rate
	pathPrefix string
	method     string
}

func createRateLimiterConfigurationSectionError(index int, section string, details string) error {
	return fmt.Errorf("invalid rate limiter configuration in section #%d: '%s', %s", index+1, section, details)
}

/**
 * Rate limit custom path format syntax:
 * <rate>:<path>:<method>
 * Examples:
 * Single rate (no path specified - targets any path):
 *	`5-M` (identical to `5-M:/`) --- 5 req per minute on any path
 * Single rate on specific path and method:
 *	`5-M:/v1/endpoint:post` --- 5 requests per minute on path starting with /v1/endpoint for method post
* Single rate on specific path:
 *	`5-M:/v1/endpoint` --- 5 requests per minute on path starting with /v1/endpoint for all methods
 * Multiple rates:
 *	`5-M:/v1/endpoint,10-M:/v2/endpoint` --- 5 requests per minute on /v1/endpoint, 10 rpm on /v2/endpoint
 * Complex scenario:
 *	`10000-H,1000-M,5-M:/v1/endpoint` --- 10000 requests per hour on any path, 1000 per minute on any path, 5 requests per minute on /v1/endpoint
*/
func parseRateLimiterConfiguration(input string) ([]RateLimiterConfiguration, error) {
	var configurations []RateLimiterConfiguration
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return configurations, nil
	}
	for index, section := range strings.Split(input, ",") {
		if len(section) == 0 {
			return nil, createRateLimiterConfigurationSectionError(index, section, "no content, expected 'rate:path' format")
		}
		ratePathAndMethod := strings.Split(section, ":")
		if len(ratePathAndMethod) > 3 {
			return nil, createRateLimiterConfigurationSectionError(index, section, "too many elements, expected 'rate:path:method' format")
		}

		rateConfig := ratePathAndMethod[0]
		rate, err := limiter.NewRateFromFormatted(rateConfig)
		if err != nil {
			return nil, createRateLimiterConfigurationSectionError(index, section, "unable to parse rate: "+err.Error())
		}
		pathPrefix := "/"
		method := ""
		if len(ratePathAndMethod) >= 2 {
			pathPrefix = ratePathAndMethod[1]
			if pathPrefix == "" {
				return nil, createRateLimiterConfigurationSectionError(index, section, "path should not be empty")
			}
			if !strings.HasPrefix(pathPrefix, "/") {
				return nil, createRateLimiterConfigurationSectionError(index, section, "path should start with /")
			}
			if path.Clean(pathPrefix) != pathPrefix {
				return nil, createRateLimiterConfigurationSectionError(index, section, "path is not clean, expected path '"+path.Clean(pathPrefix)+"'")
			}
			if len(ratePathAndMethod) == 3 {
				method = strings.ToUpper(ratePathAndMethod[2])
				if method != http.MethodPost && method != http.MethodGet && method != http.MethodPatch && method != http.MethodDelete && method != http.MethodPut {
					return nil, createRateLimiterConfigurationSectionError(index, section, "method '"+method+"' is not valid. Allowed methods are GET/PUT/POST/PATCH/DELETE")
				}
			}
		}
		configurations = append(configurations, RateLimiterConfiguration{
			rate:       rate,
			pathPrefix: pathPrefix,
			method:     method,
		})
	}
	return configurations, nil
}

func initRateLimiters(ctx context.Context, options *Options) ([]filters.RateLimiterMiddleware, error) {
	var rateLimiters []filters.RateLimiterMiddleware
	if !options.APISettings.RateLimitingEnabled {
		return nil, nil
	}

	var redisStore limiter.Store
	var err error
	if options.RedisClient != nil {
		redisStore, err = redis.NewStore(options.RedisClient)
		if err != nil {
			log.C(ctx).Errorf("failed to initialize redis store: %v", err)
			return nil, err
		}
	}

	configurations, err := parseRateLimiterConfiguration(options.APISettings.RateLimit)
	if err != nil {
		return nil, err
	}
	for _, configuration := range configurations {
		store := redisStore
		if store == nil {
			log.C(ctx).Error("redis client is not initialized. creating in memory store for rate limiting")
			store = memory.NewStore()
		}
		rateLimiters = append(
			rateLimiters,
			filters.NewRateLimiterMiddleware(stdlib.NewMiddleware(limiter.New(store, configuration.rate)), configuration.pathPrefix, configuration.method, configuration.rate),
		)
	}
	return rateLimiters, nil
}
