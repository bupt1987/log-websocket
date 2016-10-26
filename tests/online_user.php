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

for ($i = 1; $i <= 10000; $i++) {
    $sStr = json_encode([
        'uid' => $i,
        'ip' => '54.85.200.215', //54.85.200.215, 219.141.227.166
        'start_time' => time() + $i,
        'end_time' => time() + $i,
    ]);
    $sFormatStr = makePack($sCategory, $sStr);
    @fwrite($socket, $sFormatStr);
    echo $sFormatStr;
    usleep(1);
}

fclose($socket);

function makePack($sCategory, $sMsg) {
    return $sCategory . ',' . $sMsg . "\n";
}
