# 8-2. DB 設計（ER Diagram）

```mermaid
erDiagram
  USERS {
    uuid id PK
    string name
    string email UNIQUE
    string passwordHash
    datetime createdAt
    datetime updatedAt
  }

  PROJECTS {
    uuid id PK
    uuid ownerId FK
    string name
    string description
    string status
    datetime createdAt
    datetime updatedAt
  }

  PROJECT_MEMBERS {
    uuid projectId FK
    uuid userId FK
    string role
    datetime joinedAt
  }

  INVITATIONS {
    uuid id PK
    uuid projectId FK
    string email
    string token
    string role
    datetime expiresAt
    datetime acceptedAt
  }

  TASKS {
    uuid id PK
    uuid projectId FK
    string title
    string description
    string status
    string priority
    uuid assigneeId
    date dueDate
    int sortOrder
    datetime createdAt
    datetime updatedAt
  }

  COMMENTS {
    uuid id PK
    uuid taskId FK
    uuid authorId FK
    string body
    datetime createdAt
    datetime updatedAt
  }

  LABELS {
    uuid id PK
    uuid projectId FK
    string name
    string color
  }

  TASK_LABELS {
    uuid taskId FK
    uuid labelId FK
  }
```
