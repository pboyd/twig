import { createConnectTransport } from "@connectrpc/connect-web";
import { Code, ConnectError } from "@connectrpc/connect";

export const transport = createConnectTransport({
  baseUrl: "/",
  fetch: (input, init) =>
    fetch(input, { ...init, credentials: "include" }).then((res) => {
      return res;
    }),
  interceptors: [
    (next) => async (req) => {
      try {
        return await next(req);
      } catch (err) {
        if (
          err instanceof ConnectError &&
          err.code === Code.Unauthenticated
        ) {
          const next = encodeURIComponent(window.location.pathname + window.location.search);
          window.location.href = `/login?next=${next}`;
        }
        throw err;
      }
    },
  ],
});
