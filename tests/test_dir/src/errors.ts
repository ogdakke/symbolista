export const AUTH_ERROR = {
  unauthorized: "AE_1",
  failedToCreateStripeCustomer: "AE_2",
  failedToCreateUser: "AE_3",
  noToken: "AE_4",
  failedToGetUser: "AE_5",
  failedToUpdateUser: "AE_6",
  failedToDeleteUser: "AE_7",
  noUserFound: "AE_8",
  failedToCheckUserExists: "AE_9",
} as const;

export class AuthError extends Error {
  error?: Error;

  constructor(
    message: string,
    public readonly code: keyof typeof AUTH_ERROR,
  ) {
    super(message);
    this.name = "AuthError";
  }
}

export class UnauthorizedError extends AuthError {
  constructor(message: string, error?: Error) {
    super(message, "unauthorized");
    this.name = "UnauthorizedError";
    this.error = error;
  }
}

export class FailedToCreateStripeCustomerError extends AuthError {
  constructor(message: string, error?: Error) {
    super(message, "failedToCreateStripeCustomer");
    this.name = "FailedToCreateStripeCustomerError";
    this.error = error;
  }
}

export class FailedToCreateUserError extends AuthError {
  constructor(message: string, error?: Error) {
    super(message, "failedToCreateUser");
    this.name = "FailedToCreateUserError";
    this.error = error;
  }
}

export class NoTokenError extends AuthError {
  constructor(message: string, error?: Error) {
    super(message, "noToken");
    this.name = "NoTokenError";
    this.error = error;
  }
}

export class FailedToGetUserError extends AuthError {
  constructor(message: string, error?: Error) {
    super(message, "failedToGetUser");
    this.name = "FailedToGetUserError";
    this.error = error;
  }
}

export class FailedToUpdateUserError extends AuthError {
  constructor(message: string, error?: Error) {
    super(message, "failedToUpdateUser");
    this.name = "FailedToUpdateUserError";
    this.error = error;
  }
}

export class FailedToDeleteUserError extends AuthError {
  constructor(message: string, error?: Error) {
    super(message, "failedToDeleteUser");
    this.name = "FailedToDeleteUserError";
    this.error = error;
  }
}

export class NoUserFoundError extends AuthError {
  constructor(message: string, error?: Error) {
    super(message, "noUserFound");
    this.name = "NoUserFoundError";
    this.error = error;
  }
}
