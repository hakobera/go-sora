# go-sora

go-sora は [WebRTC SFU Sora](https://sora.shiguredo.jp/) の go クライアントライブラリです。主にシグナリング部分の処理を実装しています。

このライブラリ Sora を開発している時雨堂が提供しているライブラリではありませんので、使用方法などについて時雨道に問い合わせはしないでください。

## 前提事項

go-sora を利用するには go 1.14 以上が必要です。

## 使い方

[SDL2 example](./examples/sdl2) を参照してください。

## 制限事項

現在のバージョンでは、以下の機能はサポートされていません。

* 送信
  * マルチストリーム
  * スポットライト
  * サイマルキャスト
* 受信
  * マルチストリーム
  * スポットライト
* HTTP API

## LICENSE

```
Copyright 2020 Kazuyuki Honda (hakobera)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```
