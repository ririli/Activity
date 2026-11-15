package main

import (
	"fmt"
	"testing"

	"activity/pkg/platform"
)

func TestRankingRedisClientModes(t *testing.T) {
	fallback := platform.Redis()
	defer fallback.Close()
	t.Setenv("RANKING_REDIS_SHARD_ADDRS", "")
	t.Setenv("RANKING_REDIS_SENTINEL_ADDRS", "")
	t.Setenv("RANKING_REDIS_PASSWORD", "")
	t.Setenv("RANKING_REDIS_SENTINEL_PASSWORD", "")
	clients, err := rankingRedisClients(fallback)
	if err != nil {
		t.Fatal(err)
	}
	for _, client := range clients {
		if client != fallback {
			t.Fatal("local mode must share one client")
		}
	}

	const seeds = "s0:26379,s1:26379,s2:26379"
	t.Setenv("RANKING_REDIS_SENTINEL_ADDRS", seeds)
	if _, err := rankingRedisClients(fallback); err == nil {
		t.Fatal("accepted missing credentials")
	}
	t.Setenv("RANKING_REDIS_PASSWORD", "data-password")
	t.Setenv("RANKING_REDIS_SENTINEL_PASSWORD", "sentinel-password")
	t.Setenv("RANKING_REDIS_SHARD_ADDRS", "r0:6379,r1:6379,r2:6379,r3:6379")
	if _, err := rankingRedisClients(fallback); err == nil {
		t.Fatal("accepted multiple modes")
	}
	t.Setenv("RANKING_REDIS_SENTINEL_ADDRS", "")
	clients, err = rankingRedisClients(fallback)
	if err != nil {
		t.Fatal(err)
	}
	for i, client := range clients {
		defer client.Close()
		if client.Options().Addr != fmt.Sprintf("r%d:6379", i) {
			t.Fatalf("shard %d got the wrong direct address: %s", i, client.Options().Addr)
		}
	}
	t.Setenv("RANKING_REDIS_SHARD_ADDRS", "")
	t.Setenv("RANKING_REDIS_SENTINEL_ADDRS", seeds)
	clients, err = rankingRedisClients(fallback)
	if err != nil {
		t.Fatal(err)
	}
	for i, client := range clients {
		defer client.Close()
		if client == fallback || client.Options().Password != "data-password" || i > 0 && client == clients[i-1] {
			t.Fatalf("shard %d did not get a dedicated failover client", i)
		}
	}
}

func TestSentinelAddresses(t *testing.T) {
	for _, raw := range []string{
		"s0:26379,s1:26379", "s0:0,s1:26379,s2:26379", "s0:65536,s1:26379,s2:26379",
		"s0:26379,S0.:026379,s2:26379", "s0:26379,:26379,s2:26379", "s0,s1:26379,s2:26379",
	} {
		if _, err := sentinelAddresses(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	if _, err := sentinelAddresses("s0:26379, 127.0.0.1:26379, [::1]:26379"); err != nil {
		t.Fatal(err)
	}
}
