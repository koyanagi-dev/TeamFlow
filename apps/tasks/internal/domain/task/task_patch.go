package task

import "time"

type TaskPatch struct {
    Title       Patch[string]
    Description Patch[string]
    Status      Patch[TaskStatus]
    Priority    Patch[TaskPriority]
    AssigneeID  Patch[string]
    DueDate     Patch[time.Time]
}

func (t *Task) ApplyPatch(p TaskPatch) error {
    // 骨格だけ。既存Taskのフィールド/メソッド名に合わせて後で調整する
    if p.Status.IsSet && p.Status.IsNull {
        return ErrInvalidPatch("status cannot be null")
    }
    if p.Priority.IsSet && p.Priority.IsNull {
        return ErrInvalidPatch("priority cannot be null")
    }
    t.TouchUpdatedAt()
    return nil
}
