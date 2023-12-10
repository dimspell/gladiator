import CardLayout from '../../components/CardLayout'
import { Link } from 'react-router-dom'

function JoinServer() {
  return (
    <CardLayout>
      <div className={'space-y-6 py-8 text-base'}>
        <div>
          <h2 className='font-bold text-gray-100 mb-2'>Join a Server</h2>
          <p className='text-gray-200'>
            Please enter the address of the server you wish to connect to:
          </p>
        </div>
        <div className={'bg-yellow-900 text-yellow-50 py-4 px-4 rounded-lg'}>
          <h3 className='font-bold'>Could not reach server </h3>
          <p className={'my-1'}>
            Unable to establish a connection to the server.
            Please check your internet connection and the URL address you have provided.
            In case of LAN server, make sure you belong to the network you are trying connect to.
          </p>
          <p className={'text-sm'}>
            <a href={'#'} className={'font-mono text-sky-400 underline'}>ERR2137</a>
            <span> (View known bugs and issues)</span>
          </p>
        </div>
        <form>
          <div className='max-w-sm w-full md:max-w-full'>
            <div className='mb-6'>
              <label
                className='block text-gray-300 font-bold mb-1 pr-4'
                htmlFor='console-address'>
                URL
              </label>
              <input
                className='bg-gray-200 appearance-none border-2 border-gray-200 rounded w-full py-2 px-4 text-gray-700 leading-tight focus:outline-none focus:bg-white focus:border-sky-500'
                id='console-address'
                type='text'
                value='http://127.0.0.1:2137' />
            </div>
            <div className='text-right'>
              <button
                className='shadow bg-sky-600 hover:bg-sky-500 focus:shadow-outline focus:outline-none text-sky-100 font-bold py-2 px-4 rounded'
                type='button'>
                Connect
              </button>
            </div>
          </div>
        </form>
        <div className='text-base leading-7'>
          <p>
            <Link to={'/'} className='text-sky-500 hover:text-sky-600'>
              &larr; Go back
            </Link>
          </p>
        </div>

      </div>
    </CardLayout>
  )
}

export default JoinServer