export interface SecurityConfig {
  authToken: string | string[];
}

export const isAuthTokenValid = (config: SecurityConfig, reqAuthToken: string): boolean => {
  let configToken = config.authToken || [''];
  if (typeof configToken === 'string') {
    configToken = [configToken];
  }

  return reqAuthToken !== '' && configToken.includes(reqAuthToken);
};
