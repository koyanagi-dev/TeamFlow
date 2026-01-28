# ADR: Tasks List Cursor-based Pagination (v1)

- Status: Accepted
- Date: 2026-01-12
- Scope: GET /api/projects/{projectId}/tasks

## Context

Tasks 一覧 API は filter/search/sort/limit をサポートしており、現状 `sort` は複数キー指定が可能である。
しかし Cursor-based pagination（seek pagination）は、順序が安定していること（決定性）と
「同一クエリ条件の続き」であること（整合性）が満たされない場合、重複・欠落・意図しない結果を生む。

v1 では、実装の複雑性（NULL ordering、enum順序、複合インデックス爆発）と事故リスクを抑え、
安全に Cursor pagination を導入することを優先する。

## Decision

### 1. Cursor 利用時の sort 制限（v1）

cursor を指定する場合、`sort` は指定不可（もしくは `createdAt` のみ許可）とし、
内部の順序は常に以下に固定する。

- ORDER BY: `created_at ASC, id ASC`
- tie-breaker: `id`（常に最後に付与し、一意に順序を確定させる）

これにより seek 条件が単純化され、重複/欠落の事故を最小化できる。

### 2. Cursor の payload（v1）

cursor はサーバが発行し、クライアントは不透明（opaque）な値として扱う。
payload は JSON を Base64URL（paddingなし）でエンコードし、HMAC 署名を付与する。

payload（JSON）:

```json
{
  "v": 1,
  "createdAt": "2026-01-10T12:34:56.123456Z",
  "id": "task_abc123",
  "projectId": "proj_xyz",
  "qhash": "4kF9xQ2m",
  "iat": 1736514896
}

・v: バージョン（必須）
・createdAt: seek の第1キー（必須）
・id: tie-breaker（必須）
・projectId: 混用防止・アクセス防止（強く推奨）
・qhash: クエリ指紋（推奨）
・iat: 発行時刻（推奨、有効期限管理に利用）

NOTE:
・mode は v1 では不要（ASC固定のため）。DESC などの対応が必要になったタイミングで追加を検討する。

3. エンコード方式と改ざん防止（必須）

cursor の文字列表現は以下とする。
<payload_b64url>.<sig_b64url>

・payload_b64url: JSON を Base64URL（RawURLEncoding）でエンコードしたもの
・sig_b64url: HMAC_SHA256(secret, payload_b64url) を Base64URL でエンコードしたもの

署名（HMAC）は必須とする。
理由:
・payload 改ざん（projectId / createdAt 等の書き換え）を防止し、サーバ発行であることを保証するため。

4. Cursor の有効期限（推奨・v1で導入）

iat を用いて cursor の有効期限をチェックする（例: 24時間）。
・now - iat > 86400 の場合、期限切れとしてエラーにする
・期限を設けることで、鍵ローテーションや長期保持による整合性問題のリスクを下げる

5. 時刻精度（必須・v1で対策）

PostgreSQL timestamptz の実質精度（マイクロ秒）と Go time.Time（ナノ秒）差により、
比較がズレる事故を避けるため、cursor で扱う createdAt はマイクロ秒に丸める。

・cursor 生成時: task.CreatedAt を Truncate(time.Microsecond) したうえで RFC3339Nano で文字列化
・cursor 復号時: 文字列 parse 後に Truncate(time.Microsecond) を適用して seek 条件に使う

6. seek 条件（v1）

ORDER BY が created_at ASC, id ASC の場合、次ページ取得の seek 条件は以下とする（排他的 after）。

WHERE:
・created_at > :createdAt
OR (created_at = :createdAt AND id > :id)

LIMIT:
・次ページの有無判定のため、limit + 1 件取得する

7. qhash（クエリ指紋）で混用を防止（推奨・v1で導入）

cursor は「同一クエリ条件の続き」である必要があるため、payload に qhash を含める。

・qhash は projectId と filter/search 等のパラメータを正規化してハッシュ化した短い文字列とする
・正規化: 複数値（status/priority 等）はソートして join（順序差を吸収）
・ハッシュ: sha256 の先頭 8byte を Base64URL など（衝突確率は実運用上無視できる）

リクエスト時に再計算した qhash と payload の qhash が一致しない場合、cursor は無効とする。

8. エラー設計（v1）

cursor 関連は 400 の validation error とし、ErrorResponse.details.issues[] に載せる。
推奨コードは以下。

・INCOMPATIBLE_WITH_CURSOR: cursor と sort の組み合わせが不許可
・INVALID_FORMAT: cursor 形式不正（payload.sig でない等）
・INVALID_SIGNATURE: 署名不一致（改ざん疑い）
・EXPIRED: 有効期限切れ
・QUERY_MISMATCH: qhash 不一致（フィルタ等が変更された）

9. インデックス（実装時の前提）

v1 のクエリを効率よく実行するため、以下の複合インデックスを用意する。
・(project_id, created_at ASC, id ASC)
soft delete がある場合は partial index を検討する。

Consequences

Pros
・v1 の実装がシンプルで、seek 条件が明確（重複/欠落の事故を減らせる）
・署名により改ざん耐性が高い
・qhash により条件混用を明示的に弾け、UXの予測可能性が上がる
・インデックス設計が単純で、メンテコストが低い

Cons / Trade-offs
・cursor 利用時の sort 自由度が下がる（複数 sort の seek は v2 以降で検討）
・cursor の期限切れにより、長期保持していた cursor は再取得が必要

Rollout Plan
・Phase 1（v1: 今回）
  1. cursor pagination の基本実装（createdAt ASC, id ASC 固定）
  2. HMAC 署名
  3. qhash 混用検出
  4. createdAt マイクロ秒丸め
  5. 有効期限チェック（例: 24h）

・Phase 2（将来）
  ・updatedAt のサポート
  ・ASC/DESC のサポート（必要な場合のみ）
```
