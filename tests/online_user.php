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

$lIp = [
    '54.85.200.215',
    '219.141.227.166',
    '10.0.0.2',
];
$iCount = count($lIp);

$iNow = time();
$fStartTime = microtime(TRUE);
for ($i = 1; $i <= 10000; $i++) {
    $sStr = json_encode([
        'uid' => $i,
        'ip' => $lIp[$i % $iCount],
        'start_time' => $iNow,
        'end_time' => $iNow + $i,
    ]);
    $sFormatStr = makePack($sCategory, $sStr);
    @fwrite($socket, $sFormatStr);
    echo $sFormatStr;
    usleep(1);
}
echo 'Cost: ' . (microtime(TRUE) - $fStartTime) . "\n";

fclose($socket);

function makePack($sCategory, $sMsg) {
    return $sCategory . ',' . $sMsg . "\n";
}
