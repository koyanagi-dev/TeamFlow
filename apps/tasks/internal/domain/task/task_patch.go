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
	if err := t.applyStatusPatch(p.Status); err != nil {
		return err
	}
	if err := t.applyPriorityPatch(p.Priority); err != nil {
		return err
	}
	if err := t.applyTitlePatch(p.Title); err != nil {
		return err
	}
	if err := t.applyDescriptionPatch(p.Description); err != nil {
		return err
	}
	if err := t.applyAssigneeIDPatch(p.AssigneeID); err != nil {
		return err
	}
	if err := t.applyDueDatePatch(p.DueDate); err != nil {
		return err
	}
	t.TouchUpdatedAt()
	return nil
}

func (t *Task) applyStatusPatch(p Patch[TaskStatus]) error {
	if !p.IsSet {
		return nil
	}
	if p.IsNull {
		return ErrInvalidPatch("status cannot be null")
	}
	if err := validateStatus(p.Value); err != nil {
		return ErrInvalidPatch(err.Error())
	}
	t.Status = p.Value
	return nil
}

func (t *Task) applyPriorityPatch(p Patch[TaskPriority]) error {
	if !p.IsSet {
		return nil
	}
	if p.IsNull {
		return ErrInvalidPatch("priority cannot be null")
	}
	if err := validatePriority(p.Value); err != nil {
		return ErrInvalidPatch(err.Error())
	}
	t.Priority = p.Value
	return nil
}

func (t *Task) applyTitlePatch(p Patch[string]) error {
	if !p.IsSet || !p.HasValue() {
		return nil
	}
	if p.Value == "" {
		return ErrInvalidPatch("task title must not be empty")
	}
	t.Title = p.Value
	return nil
}

func (t *Task) applyDescriptionPatch(p Patch[string]) error {
	if !p.IsSet {
		return nil
	}
	if p.IsNull {
		t.Description = ""
	} else {
		t.Description = p.Value
	}
	return nil
}

func (t *Task) applyAssigneeIDPatch(p Patch[string]) error {
	if !p.IsSet {
		return nil
	}
	if p.IsNull {
		t.AssigneeID = nil
	} else {
		t.AssigneeID = &p.Value
	}
	return nil
}

func (t *Task) applyDueDatePatch(p Patch[time.Time]) error {
	if !p.IsSet {
		return nil
	}
	if p.IsNull {
		t.DueDate = nil
	} else {
		t.DueDate = &p.Value
	}
	return nil
}
