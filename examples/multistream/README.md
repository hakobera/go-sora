# SDL2 マルチストリーム

go-sora と Pion、libvpx を使って、WebRTC SFU Sora 経由で受け取ったマルチストリームの Video データを [SDL2](https://www.libsdl.org/) を使って表示するサンプルコードです。

## 既知の問題

* 長時間受信し続けると、以下のようなエラーが発生することがあります。特に受信する動画の解像度が高い場合、もしくは送信側でビットレート指定をした場合に頻発するので、問題が発生した場合は、解像度を下げたり、ビットレート指定を外すことを推奨します。
  * 動画の表示が乱れる
  * `Failed to process video frame: Failed to decode frame. Corrupt frame detected : Keyframe / intra-only frame required to reset decoder state` というエラーが出て、動画が止まる
  * メモリアクセス例外が発生する

この問題は Sora もしくは go-sora 自体ではなく、[VPX Decoder](https://github.com/hakobera/go-webrtc-decoder) の実装による問題です。

## 使い方

### 依存ライブラリのインストール

`Ubuntu 20.04 LTS + libvpx 1.8` の組み合わせでのみ動作確認しています。
macOS でも動くはずですが、検証していません。Windows 環境では動作しません。
それぞれの環境で以下のコマンドを実行して、依存ライブラリをインストールしてください。

#### macOS

```console
$ brew install sdl2 libvpx
```

[Homebrew](https://brew.sh) がセットアップされていない場合は、先にセットアップを済ませておいてください。

#### Ubuntu 20.04

```console
$ sudo apt install libsdl2-dev libvpx-dev
```

### WebRTC SFU Sora の準備

WebRTC SFU Sora は各自用意してください。
これ以降は、Sora の検証サービスである [Sora Labo](https://sora-labo.shiguredo.jp/) を利用する想定で話を進めます。

### Sora Labo のオンラインサンプル「マルチストリーム送信」を開きます

Sora のオンラインサンプル 「マルチストリーム送信」をブラウザで複数タブで開き、接続したいチャンネルIDを入力し、動画のエンコード方式に `VP8` を選択して、connect ボタンを押します。

### sdl2 を実行します

上記で入力した RoomID をコマンドラインパラメータとして指定します。

```console
go run . -url wss://sora-labo.shiguredo.jp/signaling -channel-id <your_github_id>@sora-labo -signaling-key <your_signaling_key> -video-codec VP8
```

プログラムが開始されると、SDLのウィンドウが開き、PeerConnection 接続が完了すると、コンソールに `Connected` と表示されます。ブラウザからの送信された動画データは、初回キーフレームを受け取った後にSDLウィンドウ内に表示されます。

プログラムを終了するには、`Ctrl+C` を押します。

## 追加オプション

### VP9 を受信する

VP9 を受信する場合は、`-video-codec VP9` を指定します。

### 詳細ログを出力する

詳細ログを出力する場合は、`-verbose` オプションを追加します。
