# 📘 テスト戦略（Test Strategy）

### TeamFlow — Testing & TDD Guidelines

---

# 1. テスト優先順位（Testing Priorities）

TeamFlow では、ドメインロジックの正確性と長期保守を最重視し、  
**以下の優先順位で TDD（Test-Driven Development）を適用する。**

## 1.1 最優先：Go マイクロサービス（ドメイン層・アプリケーション層）

- 対象サービス：Projects / Tasks / Notifications
- テスト対象：
  - エンティティ
  - 値オブジェクト
  - ドメインサービス
  - アプリケーションサービス（ユースケースロジック）
- 目的：**ビジネスロジックを最も堅牢に保つ**

## 1.2 次点：認証サービス（Node.js / TypeScript）

- パスワードハッシュ
- JWT署名・検証
- Refresh Token ローテーション
- セッション管理

## 1.3 REST API 層の統合テスト（OpenAPI ベース）

- OpenAPI スキーマを単一の真実（Single Source of Truth）とする
- スキーマ検証を通じて API の入出力の正確性を担保する

## 1.4 フロントエンド（Next.js）の UI / E2E テスト

- コンポーネントテスト（Vitest + Testing Library）
- E2E テスト（Playwright）
- 重要な操作フローを中心に後半フェーズで実施する

---

# 2. Go サービスのテスト構成（Projects / Tasks / Notifications）

## 2.1 ディレクトリ構成

Go のテストは `internal/` 以下に次のように配置する：

```
/internal
  /domain
    entity.go
    entity_test.go

  /usecase
    create_task.go
    create_task_test.go

  /infrastructure
    repository.go
    memory_repository.go (インメモリ実装)
    memory_repository_test.go
```

## 2.2 テスト技法

- **テーブル駆動テスト**を標準化
- インフラ層は **interface で抽象化**
- TDD では **インメモリリポジトリを使用**（DB依存を排除）
- 実 DB を用いた統合テストは後半で必要最小限に限定

---

# 3. 認証サービス（Node.js / TypeScript）のテスト方針

## 3.1 テスト基盤

- Jest または Vitest
- テストファイルは `*.test.ts` とする

## 3.2 テスト対象（TDD 優先）

- パスワードハッシュ
- JWT署名／検証
- Refresh Token のローテーション
- セッション管理
- 外部サービスへの依存はモック化

---

# 4. API テスト（OpenAPI に基づく）

## 4.1 原則

OpenAPI 仕様書（`/docs/api/teamflow-openapi.yaml`）を **単一のソース・オブ・トゥルース**とする。

## 4.2 実施内容

- 型生成（TS / Go）
- リクエスト／レスポンスのスキーマ検証
- 主要ユースケースの API 統合テスト

---

# 5. フロントエンド（Next.js）のテスト

## 5.1 テスト構成

- コンポーネント：Vitest + React Testing Library
- E2E：Playwright
- テストは後半フェーズで導入する

## 5.2 重点フロー

- タスク作成・更新・完了
- コメント投稿
- プロジェクト招待
- 認証フロー（ログイン／ログアウト）

---

# 6. TDD の進め方（Red → Green → Refactor）

TDD は次のサイクルを徹底する：

1. **Red**：失敗するテストを書く
2. **Green**：最小限の実装でテストを通す
3. **Refactor**：重複排除・責務整理・命名改善
4. 次のユースケースへ進む

---

# 7. 補足

- テストは **「ユースケース単位」** で追加していく
- ドメイン層の安定が、アプリ全体の品質と速度アップに直結する
- 重要な変更は必ず「テスト → 実装」の順番で行う
