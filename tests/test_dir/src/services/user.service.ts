import { RequestEventCommon } from "@builder.io/qwik-city";
import { Logger } from "pino";
import { IAuthClient } from "~/features/authentication/auth.client";
import { UsersService } from "~/features/authentication/users.service";
import { User } from "~/models";
import { BaseService } from "~/models/service";
import { Result } from "~/utils/result";

export interface TokenServiceDeps {
  authClient: IAuthClient;
  logger: Logger;
  usersService: UsersService;
}

export class TokenService extends BaseService {
  static instance: TokenService;
  #authClient: IAuthClient;
  #usersService: UsersService;

  constructor(deps: TokenServiceDeps) {
    super({ logger: deps.logger });
    this.#authClient = deps.authClient;
    this.#usersService = deps.usersService;
  }

  static getInstance(): TokenService {
    return TokenService.instance;
  }

  async getToken() {
    const result = await this.#authClient.getSession();
    return result.data?.access_token;
  }

  async getUser({
    token,
    event,
  }: {
    token?: string;
    event?: {
      sharedMap: RequestEventCommon["sharedMap"];
    };
  }): Promise<User | undefined> {
    const result = await this.#authClient.getUser(token);
    if (Result.isErr(result)) {
      return undefined;
    }

    const storedUser = event?.sharedMap.get("user") as User | undefined;
    if (storedUser && storedUser.user.id === result.data.id) {
      return storedUser;
    }

    const user = await this.#usersService.getUserById(result.data.id);

    if (Result.isErr(user)) {
      return undefined;
    }

    const res: User = user.data;
    res.user = result.data;

    event?.sharedMap.set("user", res);
    return res;
  }

  static authHeaders(token: string) {
    return { Authorization: `Bearer ${token}` };
  }
}
