<?php
/**
 * Description: msg 格式 category,msg\n
 */
$sFluentSock = '/tmp/log-stock.socket';
$sCategory = 'ip_to_iso';

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

for ($i = 1; $i <= 1000000; $i++) {
    $sStr = $lIp[$i % $iCount];
    $sFormatStr = makePack($sCategory, $sStr);
    echo "send => " . $sFormatStr;
    @fwrite($socket, $sFormatStr);
    if ($err = error_get_last()) {
        print_r($err);
    } else {
        $res = fgets($socket);
        echo "rec => " . trim($res) . "\n";
    }
    usleep(1);
}

fclose($socket);

function makePack($sCategory, $sMsg) {
    return $sCategory . ',' . $sMsg . "\n";
}
