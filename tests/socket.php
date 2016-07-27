<?php
/**
 * Description:
 */
$sFluentSock = '/tmp/log-stock.socket';
$socket = @stream_socket_client('unix://' . $sFluentSock, $errno, $errstr, 3, \STREAM_CLIENT_CONNECT | \STREAM_CLIENT_PERSISTENT);

$sKey = "socket_test";
$sStr = json_encode([
        'test' => time(),
        'hello world' => [
            'time' => 11133322,
        ]
    ]) . "\n";
if ($socket) {
    fwrite($socket, $sStr);
    echo $sStr;
} else {
    echo "error: " . $errstr . "\n";
}
