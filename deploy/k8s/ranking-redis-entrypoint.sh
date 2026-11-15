#!/bin/sh
set -eu
umask 077

check_password() {
    case "$1" in
        ''|*[!a-zA-Z0-9_.-]*)
            echo 'Redis passwords must use letters, digits, dots, underscores or hyphens' >&2
            exit 1
            ;;
    esac
}

check_password "$RANKING_REDIS_PASSWORD"
case "$1" in
    data)
        config=/data/redis.conf
        if [ ! -f "$config" ]; then
            cat >"$config.tmp" <<EOF
port 6379
dir /data
requirepass $RANKING_REDIS_PASSWORD
masterauth $RANKING_REDIS_PASSWORD
replica-announce-ip $REDIS_NODE_HOST
replica-announce-port 6379
appendonly yes
appendfsync everysec
maxmemory 96mb
maxmemory-policy noeviction
EOF
            if [ "$REDIS_NODE_HOST" != "$REDIS_INITIAL_PRIMARY" ]; then
                printf 'replicaof %s 6379\n' "$REDIS_INITIAL_PRIMARY" >>"$config.tmp"
            fi
            mv "$config.tmp" "$config"
        fi
        exec redis-server "$config"
        ;;
    sentinel)
        check_password "$RANKING_REDIS_SENTINEL_PASSWORD"
        config=/data/sentinel.conf
        if [ ! -f "$config" ]; then
            set -- $REDIS_INITIAL_PRIMARIES
            [ "$#" -eq 4 ]
            for host in "$@"; do
                until getent hosts "$host" >/dev/null 2>&1; do sleep 1; done
            done
            cat >"$config.tmp" <<EOF
port 26379
dir /data
requirepass $RANKING_REDIS_SENTINEL_PASSWORD
sentinel resolve-hostnames yes
sentinel announce-hostnames yes
sentinel announce-ip $REDIS_NODE_HOST
sentinel announce-port 26379
EOF
            shard=0
            for host in "$@"; do
                group=ranking-shard-$shard
                cat >>"$config.tmp" <<EOF
sentinel monitor $group $host 6379 2
sentinel auth-pass $group $RANKING_REDIS_PASSWORD
sentinel down-after-milliseconds $group 5000
sentinel failover-timeout $group 60000
sentinel parallel-syncs $group 1
EOF
                shard=$((shard + 1))
            done
            mv "$config.tmp" "$config"
        fi
        exec redis-server "$config" --sentinel
        ;;
    *) exit 2 ;;
esac
