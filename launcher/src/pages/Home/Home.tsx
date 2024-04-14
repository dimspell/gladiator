import CardLayout from '../../components/CardLayout'
import { Link } from 'react-router-dom'
import { Command } from '@tauri-apps/api/shell'


function Home() {
  return (
    <CardLayout>
      <div className='divide-y divide-gray-700/50'>
        <div className='space-y-6 py-8 text-base leading-7 text-gray-300'>
          <h2 className='font-bold text-gray-100'>Greetings, brave adventurer!</h2>
          <p className='text-gray-100'>
            Whether you're stepping into the mystical realms of Dman for the first time or returning for another
            epic journey, we're thrilled to have you here. Prepare yourself for a world of magic, challenges, and
            camaraderie.
          </p>
          <h2 className='font-bold text-gray-100'>Ready to Begin Your Journey?</h2>
          <p className='text-gray-100'>
            Follow the wizard to host your very own server or choose an existing server to join forces and forge
            alliances as you embark on quests together.
          </p>
          <div className={'flex flex-row space-x-7'}>
            <Link to={'join-server'}
                  className='flex-1 text-center text-sky-100 hover:text-sky-200 bg-sky-700 hover:bg-sky-800 px-5 py-4 rounded'>
              Join a Server
            </Link>
            <Link to={'host-server'}
                  className='flex-1 text-center text-amber-100 hover:text-amber-200 bg-amber-700 hover:bg-amber-800 px-5 py-4 rounded'>
              Host a Server
            </Link>
          </div>
        </div>
        <div className='pt-8 text-base font-semibold leading-7'>
          <p className='text-gray-100'>
            Are you curious about the development?
          </p>
          <p>
            <a href='https://tailwindcss.com/docs'
               className='text-sky-500 hover:text-sky-600'
               onClick={async (event) => {
                 event.preventDefault()
                 console.log('clicked')


                 const command = new Command('console-memory',
                   ['console',
                     '--console-addr', '127.0.0.1:2137',
                     '--database-type', 'memory'])

                 // await child.write('message');

                 command.on('close', data => {
                   console.log(`command finished with code ${data.code} and signal ${data.signal}`)
                 })

                 command.on('error', error => console.error(`command error: "${error}"`))
                 command.stdout.on('data', line => console.log(`command stdout: "${line}"`))
                 command.stderr.on('data', line => console.log(`command stderr: "${line}"`))

                 const child = await command.spawn()

                 console.log(child.pid)

                 setTimeout(async () => {
                   console.log('killing')
                   console.log(await child.kill())
                 }, 3000)


                 // console.log('invoked:', await invoke('run_background'))
               }}>
              Join Discord channel &rarr;
            </a>
          </p>
        </div>
      </div>
    </CardLayout>
  )
}

export default Home