<?php
/**
 * Description:
 */
$sFluentSock = '/tmp/log-stock.socket';

for ($i = 0; $i < 100000; $i++) {
    $socket = @stream_socket_client('unix://' . $sFluentSock, $errno, $errstr, 3, \STREAM_CLIENT_CONNECT | \STREAM_CLIENT_PERSISTENT);
    $sStr = json_encode([
            'test' => time(),
            'hello world' => [
                'time' => 11133322,
            ],
        ]) . "\n";
    if ($socket) {
        @fwrite($socket, $sStr);
        echo $sStr;
        fclose($socket);
        usleep(1);
    } else {
        echo "error: " . $errstr . "\n";
    }
}
