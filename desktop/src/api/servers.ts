import { apiGet } from "./client";
import type { Server, ServersResponse } from "../types/server";

export async function getServers(): Promise<Server[]> {
  const response = await apiGet<ServersResponse>("/servers");
  return response.data;
}
