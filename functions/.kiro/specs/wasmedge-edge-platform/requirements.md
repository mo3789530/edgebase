# 要件定義書

## はじめに

WasmEdgeランタイムを用いて、エッジノード群（POP）上で多量の短命HTTPハンドラ関数（WASMモジュール）を低遅延かつ安全に実行するプラットフォームを構築します。設計はPullベース（Control Planeが通知 → Edgeが取得）を基本とし、エッジ側は軽量なGo実装のRunnerを用います。

## 用語集

- **CP**: Control Plane - 関数管理、ビルド、Artifact保管、通知を担当するコントロールプレーン
- **Edge Node**: POP上で実行されるエージェント（HTTPサーバ + WasmEdge埋め込み）
- **Runner**: Edge Node上で動作するGo実装の実行エージェント
- **Artifact Store**: MinIO/S3互換ストレージ（WASMバイナリ格納）
- **Function**: デプロイ対象のWASMモジュール（メタ情報を伴う）
- **Route**: HTTPパスからFunctionへのマッピング
- **Hot Instance**: メモリに常駐して何度も再呼び出しするWasmインスタンス
- **Cold Start**: インスタンス生成から初回呼び出しまでの処理
- **WASI**: WebAssembly System Interface - WASMモジュールがホスト機能にアクセスするための標準インターフェース
- **POP**: Point of Presence - エッジノードが配置される地理的拠点

## 要件

### 要件 1

**ユーザーストーリー:** 開発者として、WASM関数をControl Planeに登録できるようにしたい。そうすることで、エッジノードで実行可能な関数を管理できる。

#### 受け入れ基準

1. WHEN 開発者がFunction登録APIにFunction名、エントリーポイント、メモリページ数、最大実行時間を含むリクエストを送信する THEN the CP SHALL Function recordをデータベースに作成し、presigned upload URLを返す
2. WHEN 開発者がArtifact Storeにビルド済みWASMファイルをアップロードする THEN the CP SHALL SHA256ハッシュを計算し、Artifact URLとともにFunction recordに保存する
3. WHEN Function登録が完了する THEN the CP SHALL Function ID、バージョン、Artifact URL、SHA256を含むFunction metadataを永続化する
4. WHEN 開発者が無効なメモリページ数（0以下または上限超過）を指定する THEN the CP SHALL 登録を拒否し、エラーメッセージを返す

### 要件 2

**ユーザーストーリー:** 開発者として、登録済みFunctionを特定のエッジノードにデプロイできるようにしたい。そうすることで、地理的に分散した環境で関数を実行できる。

#### 受け入れ基準

1. WHEN 開発者がデプロイAPIにFunction ID、バージョン、ターゲットPOPを指定してリクエストを送信する THEN the CP SHALL 対象Edge Nodeを選択し、デプロイ通知をキューに追加する
2. WHEN デプロイ通知が作成される THEN the CP SHALL Function ID、バージョン、Artifact URL、SHA256、メモリページ数、最大実行時間を通知に含める
3. WHEN Edge NodeがCPに接続している THEN the CP SHALL gRPCまたはWebSocketを通じてデプロイ通知をEdge Nodeに送信する
4. WHEN デプロイ通知の送信に失敗する THEN the CP SHALL 指数バックオフで再試行し、通知を永続キューに保持する

### 要件 3

**ユーザーストーリー:** Edge Nodeとして、CPからデプロイ通知を受信し、WASM Artifactを取得できるようにしたい。そうすることで、最新の関数を実行できる。

#### 受け入れ基準

1. WHEN Edge NodeがCPからデプロイ通知を受信する THEN the Edge Node SHALL Artifact URLからWASMファイルをダウンロードする
2. WHEN WASMファイルのダウンロードが完了する THEN the Edge Node SHALL SHA256ハッシュを計算し、通知に含まれるSHA256と比較する
3. WHEN SHA256ハッシュが一致する THEN the Edge Node SHALL WASMファイルをローカルキャッシュに保存し、CPにデプロイ成功ステータスを報告する
4. WHEN SHA256ハッシュが一致しない THEN the Edge Node SHALL WASMファイルを破棄し、CPにデプロイ失敗ステータスを報告する
5. WHEN Artifact Storeへの接続に失敗する THEN the Edge Node SHALL 指数バックオフで再試行し、最大試行回数後にCPにエラーを報告する

### 要件 4

**ユーザーストーリー:** Edge Nodeとして、HTTPリクエストを受信し、適切なWASM関数にルーティングできるようにしたい。そうすることで、エンドユーザーのリクエストを処理できる。

#### 受け入れ基準

1. WHEN Edge NodeがHTTPリクエストを受信する THEN the Edge Node SHALL ホスト名とパスに基づいてRouteテーブルを検索し、対応するFunction IDを特定する
2. WHEN 対応するRouteが見つかる THEN the Edge Node SHALL Function IDに対応するWASMモジュールがローカルキャッシュに存在するか確認する
3. WHEN WASMモジュールがローカルキャッシュに存在しない THEN the Edge Node SHALL Artifact Storeから同期的にWASMモジュールを取得し、キャッシュに保存する
4. WHEN 対応するRouteが見つからない THEN the Edge Node SHALL HTTP 404ステータスコードを返す
5. WHEN HTTPリクエストのパースに失敗する THEN the Edge Node SHALL HTTP 400ステータスコードを返す

### 要件 5

**ユーザーストーリー:** Edge Nodeとして、WASM関数を効率的に実行できるようにしたい。そうすることで、低レイテンシでリクエストを処理できる。

#### 受け入れ基準

1. WHEN Edge NodeがWASM関数を実行する必要がある THEN the Edge Node SHALL Hot Instance Poolに利用可能なインスタンスが存在するか確認する
2. WHEN Hot Instance Poolに利用可能なインスタンスが存在する THEN the Edge Node SHALL 既存のインスタンスを再利用し、1ミリ秒未満のオーバーヘッドで関数を呼び出す
3. WHEN Hot Instance Poolに利用可能なインスタンスが存在しない THEN the Edge Node SHALL 新しいWasmEdgeインスタンスを作成し、Cold Startを実行する
4. WHEN WASM関数の実行が完了する THEN the Edge Node SHALL インスタンスをHot Instance Poolに戻すか、プールサイズが上限に達している場合は破棄する
5. WHEN Hot Instance Poolのメモリ使用量が閾値を超える THEN the Edge Node SHALL LRUポリシーに基づいて最も古いインスタンスを破棄する

### 要件 6

**ユーザーストーリー:** Edge Nodeとして、WASM関数の実行を制限できるようにしたい。そうすることで、悪意のある関数やバグのある関数からシステムを保護できる。

#### 受け入れ基準

1. WHEN Edge NodeがWASM関数を実行する THEN the Edge Node SHALL Function metadataに指定されたメモリページ数の上限を適用する
2. WHEN WASM関数の実行時間がmax_execution_msを超える THEN the Edge Node SHALL 実行をキャンセルし、HTTP 504ステータスコードを返す
3. WHEN WASM関数が許可されていないWASI機能（raw socket、filesystem書き込み、process spawn）にアクセスしようとする THEN the Edge Node SHALL アクセスを拒否し、エラーを返す
4. WHEN WASM関数が許可されたWASI機能（logging、clocks、random）にアクセスする THEN the Edge Node SHALL アクセスを許可し、ホスト関数を提供する
5. WHEN WASM関数がメモリページ上限を超えてメモリを割り当てようとする THEN the Edge Node SHALL 割り当てを拒否し、関数実行を終了する

### 要件 7

**ユーザーストーリー:** Edge Nodeとして、定期的にCPにヘルスステータスを報告できるようにしたい。そうすることで、CPがエッジノードの状態を監視できる。

#### 受け入れ基準

1. WHEN Edge Nodeが起動する THEN the Edge Node SHALL 30秒間隔でCPにHeartbeatリクエストを送信するタイマーを開始する
2. WHEN HeartbeatリクエストをCPに送信する THEN the Edge Node SHALL Node ID、ステータス（online/degraded/offline）、CPU使用率、メモリ使用量、キャッシュされたFunction一覧を含める
3. WHEN CPがHeartbeatレスポンスで未配信のデプロイ通知を返す THEN the Edge Node SHALL 非同期にデプロイ通知を処理し、Artifactを取得する
4. WHEN Heartbeatリクエストの送信に失敗する THEN the Edge Node SHALL 指数バックオフで再試行し、接続が回復するまで継続する

### 要件 8

**ユーザーストーリー:** 運用者として、Edge Nodeのメトリクスを収集できるようにしたい。そうすることで、システムのパフォーマンスと健全性を監視できる。

#### 受け入れ基準

1. WHEN Edge Nodeが起動する THEN the Edge Node SHALL Prometheusメトリクスエンドポイント（:9090/metrics）を公開する
2. WHEN WASM関数が呼び出される THEN the Edge Node SHALL wasm_invoke_count_totalカウンターをFunction IDごとにインクリメントする
3. WHEN WASM関数の実行が完了する THEN the Edge Node SHALL wasm_invoke_latency_seconds_bucketヒストグラムに実行時間を記録する
4. WHEN WASM関数の実行がエラーで終了する THEN the Edge Node SHALL wasm_invoke_errors_totalカウンターをFunction IDとエラーコードごとにインクリメントする
5. WHEN ローカルキャッシュからWASMモジュールが取得される THEN the Edge Node SHALL wasm_cache_hits_totalカウンターをインクリメントする
6. WHEN ローカルキャッシュにWASMモジュールが存在せずArtifact Storeから取得される THEN the Edge Node SHALL wasm_cache_misses_totalカウンターをインクリメントする

### 要件 9

**ユーザーストーリー:** 開発者として、HTTPリクエストをWASM関数にマッピングするRouteを定義できるようにしたい。そうすることで、柔軟なルーティング設定を実現できる。

#### 受け入れ基準

1. WHEN 開発者がRoute作成APIにホスト名、パスパターン、Function ID、HTTPメソッドを指定してリクエストを送信する THEN the CP SHALL Routeレコードをデータベースに作成する
2. WHEN 複数のRouteが同じホスト名とパスにマッチする THEN the CP SHALL 優先度（priority）フィールドに基づいて最も高い優先度のRouteを選択する
3. WHEN RouteにPOP selectorが指定されている THEN the CP SHALL 条件に一致するEdge Nodeにのみルーティング情報を配信する
4. WHEN Routeが作成または更新される THEN the CP SHALL 影響を受けるEdge Nodeにルーティングテーブル更新通知を送信する

### 要件 10

**ユーザーストーリー:** 開発者として、ソースコードからWASMモジュールをビルドできるようにしたい。そうすることで、手動でビルドする手間を省ける。

#### 受け入れ基準

1. WHEN 開発者がGitリポジトリをCPに登録する THEN the CP SHALL webhookを設定し、pushイベントを受信できるようにする
2. WHEN CPがGit pushイベントを受信する THEN the CP SHALL Build Workerを起動し、ソースコードをチェックアウトする
3. WHEN Build Workerがソースコードをビルドする THEN the Build Worker SHALL 言語に応じた適切なツールチェーン（Rust toolchain、wasm-pack、esbuild）を使用する
4. WHEN ビルドが成功する THEN the Build Worker SHALL WASMファイルのSHA256ハッシュを計算し、Artifact Storeにアップロードする
5. WHEN ビルドが失敗する THEN the Build Worker SHALL エラーログをデータベースに保存し、開発者に通知する

### 要件 11

**ユーザーストーリー:** 開発者として、WASM関数のデプロイをCanary方式で段階的にロールアウトできるようにしたい。そうすることで、新バージョンのリスクを最小化できる。

#### 受け入れ基準

1. WHEN 開発者がCanaryデプロイを指定する THEN the CP SHALL 最初に1つのEdge Nodeにのみ新バージョンをデプロイする
2. WHEN Canary Edge Nodeでのデプロイが成功し、メトリクスが正常である THEN the CP SHALL 段階的に残りのEdge Nodeに新バージョンをデプロイする
3. WHEN Canary Edge Nodeでエラー率が閾値を超える THEN the CP SHALL ロールアウトを停止し、開発者に通知する
4. WHEN ロールバックが要求される THEN the CP SHALL 全Edge Nodeに前バージョンへの切り替え指示を送信する

### 要件 12

**ユーザーストーリー:** セキュリティ管理者として、CPとEdge Node間の通信を暗号化できるようにしたい。そうすることで、中間者攻撃を防止できる。

#### 受け入れ基準

1. WHEN Edge NodeがCPに接続する THEN the Edge Node SHALL mTLSハンドシェイクを実行し、ノード証明書を提示する
2. WHEN CPがEdge Nodeからの接続を受け入れる THEN the CP SHALL ノード証明書を検証し、Node IDが許可リストに存在することを確認する
3. WHEN mTLS検証が成功する THEN the CP SHALL 長期有効なJWTトークンをEdge Nodeに発行する
4. WHEN Edge NodeがCP APIを呼び出す THEN the Edge Node SHALL gRPCメタデータまたはHTTPヘッダーにJWTトークンを含める
5. WHEN mTLS検証が失敗する THEN the CP SHALL 接続を拒否し、監査ログに記録する

### 要件 13

**ユーザーストーリー:** 開発者として、WASM関数から構造化ログを出力できるようにしたい。そうすることで、関数の動作をデバッグできる。

#### 受け入れ基準

1. WHEN WASM関数がlogホスト関数を呼び出す THEN the Edge Node SHALL ログレベル、メッセージ、Function ID、Request ID、タイムスタンプを含む構造化JSONログを出力する
2. WHEN Edge Nodeがログを出力する THEN the Edge Node SHALL ログを標準出力に書き込み、ログ収集システム（Loki）が取得できるようにする
3. WHEN WASM関数が許可されたログレベル（DEBUG、INFO、WARN、ERROR）でログを出力する THEN the Edge Node SHALL ログを記録する
4. WHEN WASM関数が1秒間に1000件を超えるログを出力しようとする THEN the Edge Node SHALL レート制限を適用し、超過分を破棄する

### 要件 14

**ユーザーストーリー:** 運用者として、Function登録、変更、デプロイの操作を監査できるようにしたい。そうすることで、セキュリティインシデントを追跡できる。

#### 受け入れ基準

1. WHEN 開発者がFunctionを登録する THEN the CP SHALL ユーザーID、Function ID、操作タイプ（create）、タイムスタンプを監査ログテーブルに記録する
2. WHEN 開発者がFunctionを更新する THEN the CP SHALL 変更前後の値を含む監査ログを記録する
3. WHEN 開発者がFunctionをデプロイする THEN the CP SHALL デプロイ先POP、バージョン、タイムスタンプを監査ログに記録する
4. WHEN 監査ログが記録される THEN the CP SHALL ログを不可逆的に保存し、削除や変更を防止する

### 要件 15

**ユーザーストーリー:** Edge Nodeとして、ローカルキャッシュを効率的に管理できるようにしたい。そうすることで、ディスク容量を節約できる。

#### 受け入れ基準

1. WHEN ローカルキャッシュのサイズが上限（例: 10GB）に達する THEN the Edge Node SHALL LRUポリシーに基づいて最も古いWASMファイルを削除する
2. WHEN ローカルキャッシュのファイル数が上限（例: 1000ファイル）に達する THEN the Edge Node SHALL LRUポリシーに基づいて最も古いWASMファイルを削除する
3. WHEN 新しいバージョンのWASMファイルがキャッシュに追加される THEN the Edge Node SHALL 古いバージョンを自動的に削除せず、LRU evictionに任せる
4. WHEN Edge Nodeが起動する THEN the Edge Node SHALL キャッシュディレクトリ内の全WASMファイルのSHA256を検証し、破損したファイルを削除する
