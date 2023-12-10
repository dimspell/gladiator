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

export default App
