<?php
/**
 * Description: msg 格式 category,msg\n
 */
$sFluentSock = '/tmp/log-stock.socket';
$sCategory = 'online_user';

$socket = @stream_socket_client('unix://' . $sFluentSock, $errno, $errstr, 3, \STREAM_CLIENT_CONNECT | \STREAM_CLIENT_PERSISTENT);

if (!$socket) {
    echo "error: " . $errstr . "\n";
    exit();
}

for ($i = 1; $i <= 1000000; $i++) {
    $sStr = json_encode([
        'uid' => $i,
        'ip' => '10.11.104.190',
        'start_time' => time() + $i * 10,
        'end_time' => time() + $i * 10,
    ]);
    $sFormatStr = makePack($sCategory, $sStr);
    @fwrite($socket, $sFormatStr);
    echo $sFormatStr;
    //这里为了测试性能特意fclose, 在实际使用中可以不fclose
    usleep(1);
}

fclose($socket);

function makePack($sCategory, $sMsg) {
    return $sCategory . ',' . $sMsg . "\n";
}
