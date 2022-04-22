import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import App from './App';
import reportWebVitals from './reportWebVitals';
import { RestfulProvider } from 'restful-react';

ReactDOM.render(
  <React.StrictMode>
    <RestfulProvider
      base={window.runtimeEnv.API_URL}
      requestOptions={() => {
        const headers = [
          ['Access-Control-Allow-Origin', window.runtimeEnv.API_URL],
          ['Content-Type', 'application/json'],
        ];
        return { headers };
      }}
    >
      <App />
    </RestfulProvider>
  </React.StrictMode>,
  document.getElementById('root')
);

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
