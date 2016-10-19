<?php
/**
 * Description: msg 格式 category,msg\n
 */
$sFluentSock = '/tmp/log-stock.socket';
$sCategory = '*';

$socket = @stream_socket_client('unix://' . $sFluentSock, $errno, $errstr, 3, \STREAM_CLIENT_CONNECT | \STREAM_CLIENT_PERSISTENT);

if (!$socket) {
    echo "error: " . $errstr . "\n";
    exit();
}

for ($i = 0; $i < 10000; $i++) {
    $sStr = json_encode([
        'test' => time(),
        'hello world' => [
            'time' => 11133322,
        ],
    ]);
    $sStr = makePack($sCategory, $sStr);
    @fwrite($socket, $sStr);
    echo $sStr;
    usleep(1);
}

fclose($socket);

function makePack($sCategory, $sMsg) {
    return $sCategory . ',' . $sMsg . "\n";
}
