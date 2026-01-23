export type ValidationIssue = {
  location?: 'query' | 'path' | 'body';
  field: string;
  code: string;
  message: string;
  rejectedValue?: unknown;
};

export type ErrorResponse = {
  error: string;
  message: string;
  details?: {
    issues?: ValidationIssue[];
  };
};

export type ApiError = {
  status: number;
  error: string;
  message: string;
  issues?: ValidationIssue[];
  raw?: unknown;
};
