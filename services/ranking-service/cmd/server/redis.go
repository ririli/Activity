package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"activity/pkg/hourly"
	rdb "github.com/redis/go-redis/v9"
)

func rankingRedisClients(fallback *rdb.Client) ([hourly.Shards]*rdb.Client, error) {
	var clients [hourly.Shards]*rdb.Client
	direct := os.Getenv("RANKING_REDIS_SHARD_ADDRS")
	sentinel := os.Getenv("RANKING_REDIS_SENTINEL_ADDRS")
	if sentinel != "" {
		if direct != "" {
			return clients, fmt.Errorf("configure either RANKING_REDIS_SHARD_ADDRS or RANKING_REDIS_SENTINEL_ADDRS")
		}
		addresses, err := sentinelAddresses(sentinel)
		if err != nil {
			return clients, err
		}
		password := os.Getenv("RANKING_REDIS_PASSWORD")
		sentinelPassword := os.Getenv("RANKING_REDIS_SENTINEL_PASSWORD")
		if strings.TrimSpace(password) == "" || strings.TrimSpace(sentinelPassword) == "" {
			return clients, fmt.Errorf("RANKING_REDIS_PASSWORD and RANKING_REDIS_SENTINEL_PASSWORD are required in Sentinel mode")
		}
		options := fallback.Options()
		for shard := range clients {
			clients[shard] = rdb.NewFailoverClient(&rdb.FailoverOptions{
				MasterName:            fmt.Sprintf("ranking-shard-%d", shard),
				SentinelAddrs:         addresses,
				Password:              password,
				SentinelPassword:      sentinelPassword,
				MaxRetries:            -1,
				DialTimeout:           options.DialTimeout,
				DialerRetries:         1,
				ReadTimeout:           options.ReadTimeout,
				WriteTimeout:          options.WriteTimeout,
				ContextTimeoutEnabled: true,
				// Renew pooled connections even if the Sentinel notification was missed.
				ConnMaxLifetime:       15 * time.Second,
				ConnMaxLifetimeJitter: 3 * time.Second,
			})
		}
		return clients, nil
	}
	if direct == "" {
		for i := range clients {
			clients[i] = fallback
		}
		return clients, nil
	}
	addresses := strings.Split(direct, ",")
	if len(addresses) != hourly.Shards {
		return clients, fmt.Errorf("RANKING_REDIS_SHARD_ADDRS requires %d addresses", hourly.Shards)
	}
	for i, address := range addresses {
		addresses[i] = strings.TrimSpace(address)
		if addresses[i] == "" {
			return clients, fmt.Errorf("empty Redis shard address at position %d", i)
		}
	}
	for i, address := range addresses {
		options := *fallback.Options()
		options.Addr = address
		clients[i] = rdb.NewClient(&options)
	}
	return clients, nil
}

func sentinelAddresses(raw string) ([]string, error) {
	addresses := strings.Split(raw, ",")
	if len(addresses) != 3 {
		return nil, fmt.Errorf("RANKING_REDIS_SENTINEL_ADDRS requires three distinct host:port addresses")
	}
	seen := make(map[string]bool, len(addresses))
	for i, address := range addresses {
		host, port, err := net.SplitHostPort(strings.TrimSpace(address))
		portNumber, portErr := strconv.ParseUint(port, 10, 16)
		if err != nil || host == "" || strings.ContainsAny(host, " \t\r\n/[]") || portErr != nil || portNumber == 0 {
			return nil, fmt.Errorf("invalid Redis Sentinel address at position %d", i)
		}
		if ip := net.ParseIP(host); ip != nil {
			host = ip.String()
		} else {
			host = strings.TrimSuffix(strings.ToLower(host), ".")
		}
		addresses[i] = net.JoinHostPort(host, strconv.FormatUint(portNumber, 10))
		if host == "" || seen[addresses[i]] {
			return nil, fmt.Errorf("RANKING_REDIS_SENTINEL_ADDRS requires three distinct host:port addresses")
		}
		seen[addresses[i]] = true
	}
	return addresses, nil
}
