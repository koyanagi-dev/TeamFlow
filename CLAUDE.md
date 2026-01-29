# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## Project Overview

TeamFlow は Go + Next.js のモノレポ構成によるタスク管理プロジェクト。pnpm + Turborepo で管理。

---

## Commands

### Go Backend

```bash
# Tests
cd apps/tasks && go test ./...      # 各サービスのユニットテスト
cd apps/projects && go test ./...
make go-test                         # 全 Go テスト（sqlc 再生成含む）
make test-integration                # 統合テスト（Docker で PostgreSQL 起動）

# Lint & Format
make lint-go                         # golangci-lint 実行
make format-go                       # goimports + go fmt 実行
make build-go                        # go build（コンパイルチェック）
make check-go                        # lint + build + test 統合
```

### Frontend

```bash
cd apps/frontend
pnpm dev        # 開発サーバー
pnpm build      # ビルド（型チェック含む）
pnpm lint       # ESLint
pnpm format     # Prettier フォーマット（書き込み）
```

### OpenAPI

```bash
make openapi-validate   # OpenAPI スキーマの検証
make openapi-diff       # 破壊的変更のチェック（vs origin/master）
```

### Monorepo (Turbo)

```bash
pnpm dev    # 全 app の dev サーバー起動
pnpm build  # 全 app のビルド
pnpm lint   # 全 app の lint
pnpm test   # 全 app のテスト
```

### Integrated Checks

```bash
make check-frontend  # Frontend: format:check + lint + build
make check-all       # OpenAPI + Go + Frontend 全チェック
```

---

## Development Setup

### Go Tools (Homebrew不使用)

```bash
# golangci-lint（静的解析）
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# goimports（インポート整理）
go install golang.org/x/tools/cmd/goimports@latest

# oasdiff（OpenAPI差分検証）
go install github.com/oasdiff/oasdiff@latest

# PATHに追加（.bashrc / .zshrc に記載推奨）
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Node.js Tools

```bash
# 依存関係インストール（prettier, eslint等）
pnpm install
```

### Pre-commit Hooks（必須）

初回セットアップ時に必ず以下を実行：

```bash
bash scripts/setup-precommit.sh
```

pre-commitは以下を自動実行します：

- Prettier（コードフォーマット）
- gitleaks（シークレット検出）
- 基本ファイルチェック（trailing whitespace、YAML構文など）

コミット時に自動実行されます。必要時は `git commit --no-verify` で回避可能ですが、CIでブロックされる可能性があります。

### Security Tools

```bash
# govulncheck（Go脆弱性チェック）
go install golang.org/x/vuln/cmd/govulncheck@latest
make govulncheck

# pnpm audit（npm依存関係脆弱性）
pnpm audit

# trivy（オプション、手動実行推奨）
brew install aquasecurity/trivy/trivy
trivy fs . --config trivy.yaml
```

trivyは包括的だがノイズが多いため、定期的な手動実行を推奨。週次CIでも実行されるが、結果は参考情報として扱う。

---

## Architecture

### Monorepo Structure

```
apps/
  tasks/       # Go - タスク管理サービス (sqlc + PostgreSQL)
  projects/    # Go - プロジェクト管理サービス
  frontend/    # Next.js 16 (App Router, React 19, Tailwind 4)
docs/
  api/teamflow-openapi.yaml  # OpenAPI 仕様（Single Source of Truth）
```

### Go Services (Clean Architecture)

```
internal/
  domain/      # エンティティ、値オブジェクト、Query Object（ビジネスルール）
  usecase/     # アプリケーションサービス（ドメイン + リポジトリのオーケストレーション）
  interface/http/  # HTTPハンドラ、リクエスト解析、レスポンスマッピング
  infrastructure/  # リポジトリ実装（SQL, memory）
  testutil/        # テスト用ユーティリティ
```

- domain は infrastructure に依存しない
- Query Object は domain 層で検索/フィルタ/ソート条件を表現
- リポジトリインターフェースは DB 詳細を隠蔽

### Frontend API Client

`apps/frontend/src/lib/api/` に共通クライアント:

- `client.ts`: `apiFetch<T>()` - 統一 fetch ラッパー
- `types.ts`: `ApiError`, `ErrorResponse`, `ValidationIssue`
- `error.ts`: `isErrorResponse()`, `normalizeApiError()`

---

## Key Patterns

### Error Handling

**Backend:**

- `errors.Is` / `errors.As` を使用（文字列比較しない）
- ValidationIssue: `{field, code, message}` 形式を維持

**Frontend:**

- `apiFetch` が throw する `ApiError` を一貫して扱う
- バリデーションエラーは `issues` 配列で表示

### Priority Sorting (重要)

priority は `high > medium > low` のビジネス順序でソート。
SQL では CASE 文で数値化:

```sql
CASE priority WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 END
```

### OpenAPI Query Parameters

- 複数値は `type: string` + カンマ区切り説明（`type: array` への変更は破壊的変更）
- 既存パラメータの type/format 変更禁止

---

## Role / Responsibility

Claude Code の担当範囲:

- 実装（Go/Next.js/TypeScript）
- テストの作成・更新・実行
- バグ修正
- 影響範囲の把握

Claude Code が「勝手に決めてはいけないこと」:

- API仕様（OpenAPI）を破壊する変更
- DBスキーマ変更（migration追加など）
- エラーフォーマット（ErrorResponse/ValidationIssue）の互換性を壊す変更
- 認証方式/セキュリティ方針の変更
- 大きなリファクタ（複数ディレクトリにまたがる整理、命名規約変更など）

上記に該当する可能性がある場合:

- まず「選択肢」と「影響範囲」と「推奨案」を提示し、承認を待つこと。

---

## GitHub Flow

- master は常にデプロイ可能・テストグリーンを維持
- 作業は feature branch で行い、PR でレビューを通してマージ
- ブランチ命名: `feat/xxx`, `fix/xxx`
- Conventional Commits: `feat(tasks): ...`, `fix(frontend): ...`
- **コミット本文（BODY）は日本語で書く**

### Issue運用ルール

**Issueを起票するときは必ず適切なIssueテンプレートを使用してください。**

- PRタイトルに必ず Issue番号を含める
  - 例：`feat: Projects→Tasks導線追加 (#XX)`
- PR本文に `Closes #XX` を書く
  - → マージ時にIssue自動クローズ
- 作業ブランチ名も Issue番号ベースにすること
  - 例：`feat/XX-projects-tasks-flow`

### PR運用ルール

**PRを作成するときは必ずPRテンプレートを使用してください。**

- PR作成前に必ずIssueが存在する
- PR本文の冒頭に `Closes #XX` を書く
- ブランチ名に Issue番号を含める
  - 例：`feat/XX-projects-tasks-flow`
- IssueなしPRは原則作らない（ドキュメント系も含む）

---

## Definition of Done

変更を出す前に必ず満たす:

- `git status` が clean
- 影響範囲のユニットテストが通る
- 既存テストを壊していない
- 既存のエラー形式・API互換性を壊していない
- lint/format が必要なら実行済み

---

## PR Description (Japanese)

```markdown
### 変更内容

- なにを、なぜ変えたか（目的）
- 互換性への影響（破壊的変更の有無）
- 影響範囲（対象API/画面/ユースケース）

### 動作確認

- 手動確認の手順と結果

### テスト結果

- 実行したコマンドと結果
```

---

## Communication Style

- 迷ったら確認する（独断しない）
- 大きめの変更は「選択肢 + 推奨 + 理由 + リスク」で相談する
- 変更提案は "最小差分" を優先する（MVP速度重視）
- **Claude Codeは常に日本語で回答する**（コード内コメントを除く）
