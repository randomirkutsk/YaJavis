#!/bin/sh
# Джарвис на колонке: запуск, остановка, удаление. Всё лежит в /data/opt/jarvisd,
# системные файлы не меняются (профиль запуска Алисы подкладывается bind-mount'ом).
#   jarvisd.sh boot       — из /etc/init.d/S98jarvisd при загрузке (до старта Алисы)
#   jarvisd.sh start|stop — запустить/остановить сам jarvisd
#   jarvisd.sh status
#   jarvisd.sh uninstall  — убрать всё и перезагрузить колонку
D=/data/opt/jarvisd
RUN=/tmp/jarvisd
[ -z "$SSL_CERT_FILE" ] && [ -f /system/vendor/quasar/ca-certificates.crt ] && export SSL_CERT_FILE=/system/vendor/quasar/ca-certificates.crt
[ -z "$SSL_CERT_FILE" ] && [ -f "$D/ca-certificates.crt" ] && export SSL_CERT_FILE="$D/ca-certificates.crt"
LAUNCH=/system/vendor/quasar/launch.json
PID=$RUN/jarvisd.pid
LOOP=$RUN/loop.pid
LOG=$RUN/jarvisd.log
INIT=/etc/init.d/S98jarvisd

mounted() { grep -q " $1 " /proc/mounts; }
running() { [ -f "$1" ] && kill -0 "$(cat "$1")" 2>/dev/null; }

mic_on() {
    # профиль запуска Алисы с нашим vqe_tap.so; если не получилось — Джарвис работает без микрофона
    [ -f "$D/no-mic" ] && return 0
    mounted "$LAUNCH" && return 0
    if "$D/jarvisd" patch-launch "$LAUNCH" "$RUN/launch.json" "$D/vqe_tap.so" "$RUN/mic.sock" 2>>"$LOG"; then
        mount --bind "$RUN/launch.json" "$LAUNCH"
    fi
}

# apply_update — jarvisd скачал и проверил новую версию в $RUN/jarvisd.next и вышел: подменяем файл.
# Старый удаляем первым — на /data нет места на две копии (~30 МБ).
apply_update() {
    [ -f "$RUN/jarvisd.next" ] || return 0
    echo "$(date) обновление: ставлю новую версию" >>"$LOG"
    rm -f "$D/jarvisd"
    if cp "$RUN/jarvisd.next" "$D/jarvisd" && chmod 755 "$D/jarvisd"; then
        rm -f "$RUN/jarvisd.next"
    else
        echo "$(date) обновление: не удалось скопировать — пробую ещё раз" >>"$LOG"
        sleep 2
        rm -f "$D/jarvisd"
        cp "$RUN/jarvisd.next" "$D/jarvisd" && chmod 755 "$D/jarvisd" && rm -f "$RUN/jarvisd.next"
    fi
    sync
}

loop() {
    # перезапуск, если jarvisd упал; лог не растёт бесконечно
    while [ ! -f "$D/disabled" ]; do
        [ -f "$LOG" ] && [ "$(wc -c < "$LOG")" -gt 1048576 ] && mv -f "$LOG" "$LOG.1"
        apply_update
        "$D/jarvisd" -data "$D" -mic "$RUN/mic.sock" >>"$LOG" 2>&1 &
        echo $! > "$PID"
        wait $!
        sleep 3
    done
}

start() {
    mkdir -p "$RUN"
    running "$LOOP" && return 0
    # отдельный сеанс: не умирает вместе с adb/ssh, из которых запущен
    start-stop-daemon -S -b -m -p "$LOOP" -x /bin/sh -- "$D/jarvisd.sh" loop
}

stop() {
    running "$LOOP" && kill "$(cat "$LOOP")" 2>/dev/null
    running "$PID" && kill "$(cat "$PID")" 2>/dev/null
    rm -f "$LOOP" "$PID"
}

case "$1" in
boot)
    [ -f "$D/disabled" ] && exit 0
    mkdir -p "$RUN"
    mic_on
    start
    ;;
start) start ;;
loop) loop ;;
stop) stop ;;
restart) stop; sleep 1; start ;;
status)
    echo "jarvisd: $(running "$PID" && echo работает || echo остановлен) ($("$D/jarvisd" version 2>/dev/null))"
    echo "микрофон: $(mounted "$LAUNCH" && echo подключён || echo нет)"
    echo "автозапуск: $([ -f "$D/disabled" ] && echo выключен || echo включён)"
    ;;
uninstall)
    stop
    touch "$D/disabled"
    while mounted "$LAUNCH"; do umount "$LAUNCH" || break; done
    mount -o remount,rw / 2>/dev/null
    rm -f "$INIT"
    mount -o remount,ro / 2>/dev/null
    rm -rf "$D" "$RUN"
    sync
    reboot
    ;;
*) echo "usage: $0 boot|start|stop|restart|status|uninstall"; exit 2 ;;
esac
