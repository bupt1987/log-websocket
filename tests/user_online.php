<?php
/**
 * Description: msg 格式 category,msg\n
 */
$sFluentSock = '/tmp/log-stock.socket';
$sCategory = 'online_user';


for ($i = 1; $i <= 100; $i++) {
    $socket = @stream_socket_client('unix://' . $sFluentSock, $errno, $errstr, 3, \STREAM_CLIENT_CONNECT | \STREAM_CLIENT_PERSISTENT);
    if ($socket) {
        $sStr = json_encode([
            'uid' => $i,
            'ip' => '81.2.69.142',
            'time' => time() + $i * 10,
        ]);
        $sFormatStr = makePack($sCategory, $sStr);
        @fwrite($socket, $sFormatStr);
        echo $sFormatStr;
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
