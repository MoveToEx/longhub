import axios from "axios";

const api = axios.create({
  baseURL: import.meta.env.VITE_BASE_URL ?? 'http://localhost:8000/',
  withCredentials: true,
});

api.interceptors.request.use(req => {
  req.headers['Authorization'] = 'Session ' + JSON.parse(localStorage.getItem('nmsl-session') ?? '""')
  return req;
})

export const fetcher = async (url: string) => {
  const response = await api.get(url);
  return response.data.data;
}

export default api;