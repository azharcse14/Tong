import http from 'k6/http';
import { sleep } from 'k6';

export const options = {
  vus: 10000,        // ১০০ জন user একসাথে
  duration: '10s', // ১০ সেকেন্ড ধরে
};

export default function () {
  http.get('http://localhost:8080/menu');
  sleep(0.1);
}