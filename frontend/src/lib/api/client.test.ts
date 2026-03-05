import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { api } from "./client";

function makeFetchMock(status: number, body: unknown) {
  return vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  });
}

describe("ApiClient", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    api.setAccessToken(null);
  });

  describe("GET", () => {
    it("sends GET with Content-Type: application/json", async () => {
      const fetchMock = makeFetchMock(200, { data: { id: "1" } });
      vi.stubGlobal("fetch", fetchMock);

      await api.get("/test");

      expect(fetchMock).toHaveBeenCalledOnce();
      const [, init] = fetchMock.mock.calls[0];
      expect(init.method).toBe("GET");
      expect((init.headers as Record<string, string>)["Content-Type"]).toBe(
        "application/json",
      );
    });

    it("includes Authorization header when token is set", async () => {
      api.setAccessToken("mytoken");
      const fetchMock = makeFetchMock(200, { data: {} });
      vi.stubGlobal("fetch", fetchMock);

      await api.get("/me");

      const [, init] = fetchMock.mock.calls[0];
      expect((init.headers as Record<string, string>)["Authorization"]).toBe(
        "Bearer mytoken",
      );
    });

    it("does not include Authorization header when no token", async () => {
      const fetchMock = makeFetchMock(200, { data: {} });
      vi.stubGlobal("fetch", fetchMock);

      await api.get("/public");

      const [, init] = fetchMock.mock.calls[0];
      expect(
        (init.headers as Record<string, string>)["Authorization"],
      ).toBeUndefined();
    });

    it("unwraps data from response envelope", async () => {
      const fetchMock = makeFetchMock(200, { data: { name: "Alice" } });
      vi.stubGlobal("fetch", fetchMock);

      const result = await api.get<{ name: string }>("/profile");
      expect(result).toEqual({ name: "Alice" });
    });

    it("returns undefined for 204 response", async () => {
      const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204 });
      vi.stubGlobal("fetch", fetchMock);

      const result = await api.delete("/item/1");
      expect(result).toBeUndefined();
    });
  });

  describe("POST", () => {
    it("sends POST with JSON-stringified body", async () => {
      const fetchMock = makeFetchMock(201, { data: { id: "2" } });
      vi.stubGlobal("fetch", fetchMock);

      await api.post("/items", { name: "Test" });

      const [, init] = fetchMock.mock.calls[0];
      expect(init.method).toBe("POST");
      expect(init.body).toBe(JSON.stringify({ name: "Test" }));
    });

    it("sends POST with no body when undefined", async () => {
      const fetchMock = makeFetchMock(201, { data: {} });
      vi.stubGlobal("fetch", fetchMock);

      await api.post("/action");

      const [, init] = fetchMock.mock.calls[0];
      expect(init.body).toBeUndefined();
    });
  });

  describe("error handling", () => {
    it("throws the error object from response body on non-ok response", async () => {
      const errorBody = {
        error: { code: "not_found", message: "Resource not found" },
      };
      const fetchMock = makeFetchMock(404, errorBody);
      vi.stubGlobal("fetch", fetchMock);

      await expect(api.get("/missing")).rejects.toMatchObject({
        code: "not_found",
        message: "Resource not found",
      });
    });

    it("throws a generic error when response body is not parseable", async () => {
      const fetchMock = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.reject(new Error("invalid json")),
      });
      vi.stubGlobal("fetch", fetchMock);

      await expect(api.get("/error")).rejects.toMatchObject({
        code: "unknown_error",
      });
    });
  });

  describe("credentials", () => {
    it("always sends credentials: include", async () => {
      const fetchMock = makeFetchMock(200, { data: {} });
      vi.stubGlobal("fetch", fetchMock);

      await api.get("/test");

      const [, init] = fetchMock.mock.calls[0];
      expect(init.credentials).toBe("include");
    });
  });
});
