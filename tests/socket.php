<?php
/**
 * Description: msg 格式 category,msg\n
 */
$sFluentSock = '/tmp/log-stock.socket';
$sCategory = '*';

for ($i = 0; $i < 1; $i++) {
    $socket = @stream_socket_client('unix://' . $sFluentSock, $errno, $errstr, 3, \STREAM_CLIENT_CONNECT | \STREAM_CLIENT_PERSISTENT);
    $sStr = json_encode([
        'test' => time(),
        'hello world' => [
            'time' => 11133322,
        ],
    ]);
    if ($socket) {
        $sStr = makePack($sCategory, $sStr);
        @fwrite($socket, $sStr);
        echo $sStr;
        //这里为了测试性能特意fclose, 在实际使用中可以不fclose
        fclose($socket);
        usleep(1);
    } else {
        echo "error: " . $errstr . "\n";
    }
}

function makePack($sCategory, $sMsg) {
    return $sCategory . ',' . $sMsg . "\n";
}
