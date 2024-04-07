import { atom, useAtom } from 'jotai'
import BasicLayout from '../../components/BasicLayout'
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


  const FormBlock = ({ children, label, htmlFor, name }: any) => {
    return (
      <div className='mb-4'>
        <label
          className="block font-bold text-gray-300 mb-2"
          htmlFor={htmlFor}
        >{name}</label>
        <div className='flex flex-row items-top w-full space-x-8 justify-between'>
          <div className='w-72'>
            {children}
          </div>
          <div className='text-gray-100'>
            {label}
          </div>
        </div>
      </div>
    )
  }



  return (
    <BasicLayout>
      <div className='relative bg-gray-700 pt-6'>
        <h1 className={'text-2xl text-gray-50 text-center'}>
          Dispel<span className={'font-bold'}>Multi</span>
        </h1>
        <div
          className='relative bg-gray-800 p-8 m-4 shadow-xl ring-1 ring-gray-100/5 rounded-lg mx-auto max-w-4xl'>
          <div className=''>
            <div className='mb-8'>
              <h2 className="text-xl font-bold text-gray-100 mb-2">Host a Server</h2>
              <p className='text-gray-100 my-2'>Let's get your game server up and running. Please fill out the following form to specify the configuration details:</p>
            </div>

            <form onSubmit={handleSubmit}>
              <FormBlock
                label={`Enter the IP address & port number to bind the game server to and listen for incomming connections. This will be the address that players connect to. Make sure the port is open through any firewalls.`}
                htmlFor='bindAddress'
                name='Server Address:'
              >
                <input
                  name='bindAddress'
                  value={form.bindAddress}
                  onChange={handleChange}
                  className="bg-gray-700 text-gray-100 border border-gray-500 p-2 w-full rounded mb-4 w-72"
                />
              </FormBlock>

              <FormBlock
                label={`Select the type of database to use, either SQLite or in-memory. Note: in-memory is only recommended for testing.`}
                htmlFor='databaseType'
                name='Database type:'
              >
                <select
                  className="bg-gray-700 text-gray-100 border border-gray-500 p-2 w-full rounded mb-4  w-72"
                  value={form.databaseType}
                  onChange={handleSelect}
                >
                  <option value={'sqlite'}>Saved on disk</option>
                  <option value={'memory'}>Stored in-memory</option>
                </select>
              </FormBlock>

              <FormBlock
                label={`Note: only when the database is saved on disk, enter the path to the database file.`}
                htmlFor='databasePath'
                name='Database path:'
              >
                <input
                  type='text'
                  name='databasePath'
                  value={form.databasePath}
                  onChange={handleChange}
                  disabled={form.databaseType === 'memory'}
                  className="bg-gray-700 text-gray-100 border border-gray-500 p-2 w-full rounded mb-4 disabled:opacity-50 w-72"
                />
              </FormBlock>

              <p className='text-gray-100 my-2'>
                Once you fill out these details, we'll get your game server initialized with the provided configuration. Players will then be able to connect using the IP address and port you specified.
              </p>


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
        </div>
      </div>

    </BasicLayout>
  )
}

export default HostServer