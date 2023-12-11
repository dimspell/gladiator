import React, { useState } from 'react'
import { invoke } from '@tauri-apps/api/tauri'
import './App.css'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import Home from './pages/Home/Home'
import HostServer from './pages/HostServer/HostServer'
import JoinServer from './pages/JoinServer/JoinServer'
import ErrorPage from './pages/ErrorPage'

const router = createBrowserRouter([
  {
    path: '/',
    element: <Home />,
    errorElement: <ErrorPage />,
  },
  {
    path: '/host-server',
    element: <HostServer />,
    errorElement: <ErrorPage />,
  },
  {
    path: '/join-server',
    element: <JoinServer />,
    errorElement: <ErrorPage />,
  },
  {
    path: '/home',
    element: <Home />,
    errorElement: <ErrorPage />,
  },
])


function App() {
  const [greetMsg, setGreetMsg] = useState('')
  const [name, setName] = useState('')

  async function greet() {
    // Learn more about Tauri commands at https://tauri.app/v1/guides/features/command
    setGreetMsg(await invoke('greet', { name }))
  }

  return (
    <RouterProvider router={router} />

    // <div className='container'>
    //   <div className='row'>
    //     <form
    //       onSubmit={(e) => {
    //         e.preventDefault()
    //         greet()
    //       }}
    //     >
    //       <input
    //         id='greet-input'
    //         onChange={(e) => setName(e.currentTarget.value)}
    //         placeholder='Enter a name...'
    //       />
    //       <button type='submit'>Greet</button>
    //     </form>
    //   </div>
    //   <p>{greetMsg}</p>
    // </div>
  )
}

function unused() {
  return (
    <>
      <ul about={'Join a Server'}>
        <li className={'step-1'}>
          <h2>Could not reach a server</h2>
          <p>show an error</p>
        </li>
        <li className={'step-2'}>
          <h2>Discovered server at 21.13.1.1:6128</h2>
          <pre>
            This server will GET {host}:/.well-known/dispel-multi.json
            //{
            //  "zerotier": {
            //    "enabled": true,
            //  }
            //}
          </pre>
          <div className={'step-2-connected-to-unknown/lan'}>
            This server is configured to use LAN network
          </div>
          <div className={'step-2-connected-to-zerotier'}>
            This server is configured to use ZeroTier network.
            <div>
              <h2>Do you have ZeroTier One installed on your computer?</h2>
              <button>
                Yes
                <div>Provide network key</div>
              </button>
              <button>
                No, help me install it
                <div>
                  Tutorial how to install zerotier
                </div>
              </button>
              <button>Back</button>
            </div>
          </div>
        </li>
      </ul>
    </>
  )
}

export default App
