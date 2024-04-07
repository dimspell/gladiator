import { atom, useAtom } from 'jotai'
import CardLayout from '../../components/CardLayout'
import { Link } from 'react-router-dom'


const formAtom = atom({
  bindAddress: '0.0.0.0:2137',
  databaseType: 'memory',
  databasePath: './dispel-multi-db.sqlite'
})

function HostServer() {
  const [form, setForm] = useAtom(formAtom)

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setForm({
      ...form,
      [e.target.name]: e.target.value
    })
  }

  const handleSelect = (e: React.ChangeEvent<HTMLSelectElement>) => {
    // (e: ) => setForm({...form, databaseType: e.target.dispatchEvent.value})

    setForm({
      ...form,
      databaseType: e.target.value
    })
  }

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()

   
    console.log(form)
    // submit form data
  }

  return (
    <CardLayout>
      <div className='font-sm my-6'>
        <p className='text-gray-100 my-2'>Let's get your game server up and running. Please fill out the following form to specify the configuration details:</p>
        <dl className='text-gray-100 my-2 space-y-2'>
          <dt className='font-bold'>Server Address:</dt>
          <dd>
            Enter the IP address & port number to bind the game server to and listen for incomming connections.
            This will be the address that players connect to.
            Make sure the port is open through any firewalls.
          </dd>

          <dt className='font-bold'>Database Type:</dt>
          <dd>
            Select the type of database to use, either SQLite or in-memory. Note: in-memory is only recommended for testing.
          </dd>
        </dl>
        <p className='text-gray-100 my-2'>
          Once you fill out these details, we'll get your game server initialized with the provided configuration.
          Players will then be able to connect using the IP address and port you specified.
        </p>
      </div>


      <div className="dark max-w-lg mx-auto">
        <form onSubmit={handleSubmit}>
          <h2 className="text-xl font-bold text-gray-100 mt-4 mb-6">Register</h2>

          <label
            className="block font-medium text-gray-300 mb-2"
            htmlFor='bindAddress'
          >Server Address:</label>
          <input
            name='bindAddress'
            value={form.bindAddress}
            onChange={handleChange}
            className="bg-gray-700 text-gray-100 border border-gray-500 p-2 w-full rounded mb-4"
          />

          <label
            className="block font-medium text-gray-300 mb-2"
            htmlFor='databaseType'
          >Database type:</label>
          <select
            className="bg-gray-700 text-gray-100 border border-gray-500 p-2 w-full rounded mb-4"
            value={form.databaseType}
            onChange={handleSelect}
          >
            <option value={'sqlite'}>Saved on disk</option>
            <option value={'memory'}>Stored in-memory</option>
          </select>

          <label
            className="block font-medium text-gray-300 mb-2"
            htmlFor='databasePath'
          >
            Database path (only when the database is saved on disk):
          </label>

          <input
            type='text'
            name='databasePath'
            value={form.databasePath}
            onChange={handleChange}
            disabled={form.databaseType === 'memory'}
            className="bg-gray-700 text-gray-100 border border-gray-500 p-2 w-full rounded mb-4 disabled:opacity-50"
          />

          <div className='flex flex-row justify-between items-center mt-4'>
            <Link to={'/'} className='text-sky-500 hover:text-sky-600 text-base leading-7'>
              &larr; Go back
            </Link>

            <button
              type='submit'
              className="bg-blue-500 text-white py-3 px-7 rounded hover:bg-blue-600"
            >
              Next
            </button>
          </div>
        </form>
      </div>
    </CardLayout>
  )
}

export default HostServer