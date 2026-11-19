export const API_URL = (
  import.meta.env.VITE_API_URL ?? "http://localhost:8080/api/v1"
).replace(/\/+$/, "");

export async function apiGet<T>(endpoint: string): Promise<T> {
  const response = await fetch(`${API_URL}${endpoint}`);

  if (!response.ok) {
    let message = `Ошибка API: ${response.status}`;

    try {
      const body = await response.json();
      if (body?.error) message = body.error;
    } catch {
      message = `Ошибка API: ${response.status}`;
    }

    throw new Error(message);
  }

  return response.json();
}
