import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './styles.css'

import UIkit from 'uikit'
import Icons from 'uikit/dist/js/uikit-icons'
// import 'uikit/dist/css/uikit.min.css'

// loads the Icon plugin
UIkit.use(Icons)

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
