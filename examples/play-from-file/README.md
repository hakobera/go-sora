# Play from file

go-sora と Pion を使って、ファイルから読み込んだ VP8 の Video データを WebRTC SFU Sora に送信するサンプルコードです。

## 使い方

### WebRTC SFU Sora の準備

WebRTC SFU Sora は各自用意してください。
これ以降は、Sora の検証サービスである [Sora Labo](https://sora-labo.shiguredo.jp/) を利用する想定で話を進めます。

### Sora Labo のオンラインサンプル「マルチストリーム受信」を開きます

Sora のオンラインサンプル 「マルチストリーム受信」をブラウザで複数タブで開き、接続したいチャンネルIDを入力し、connect ボタンを押します。

### play-from-file を実行します

上記で入力した RoomID をコマンドラインパラメータとして指定します。コンソールを2つ開き、別々の `input` オプションで別のファイルを指定します。

```console
go run . -url wss://sora-labo.shiguredo.jp/signaling -channel-id <your_github_id>@sora-labo -signaling-key <your_signaling_key> -input sample1.ivf
```

```console
go run . -url wss://sora-labo.shiguredo.jp/signaling -channel-id <your_github_id>@sora-labo -signaling-key <your_signaling_key> -input sample2.ivf
```

接続に成功すると、Sora Labo 上に2つの動画が表示されます。
プログラムを終了するには、`Ctrl+C` を押します。

## サンプル動画について

サンプル動画は [Mixkit](https://mixkit.co/) からダウンロードしたものを IVF 形式に変換したものです。
オリジナルの動画は以下の URL から入手できます。

sample1: https://mixkit.co/free-stock-video/clouds-and-blue-sky-2408/
sample2: https://mixkit.co/free-stock-video/traffic-in-an-underground-tunnel-4067/