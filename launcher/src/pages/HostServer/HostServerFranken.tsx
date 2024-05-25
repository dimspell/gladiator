import { atom, useAtom } from 'jotai/index'
import { invoke } from '@tauri-apps/api'
import { Link } from 'react-router-dom'

const formAtom = atom({
  bindAddress: '0.0.0.0:2137',
  databaseType: 'memory',
  databasePath: './dispel-multi-db.sqlite',
})

function HostServerFranken() {
  const [form, setForm] = useAtom(formAtom)

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setForm({
      ...form,
      [e.target.name]: e.target.value,
    })
  }

  const handleSelect = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setForm({
      ...form,
      databaseType: e.target.value,
    })
  }

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()

    invoke('run_background')

    console.log(form)
    // submit form data
  }


  return (
    <div className=''>
      <div className='h-screen grid'>
        <div className='col-span-1 h-24 flex flex-col justify-between bg-zinc-900 p-8 text-white'>
          <div className='flex items-center text-lg font-medium'>
            <svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='currentColor'
                 strokeWidth='2' strokeLinecap='round' strokeLinejoin='round' className='mr-2 h-6 w-6'>
              <path d='M15 6v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3V6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3'></path>
            </svg>
            <h1>Dispel<span className={'font-bold'}>Multi</span></h1>
          </div>

        </div>

        <div className='max-w-xl mx-auto my-4 space-y-6'>

          <div className={''}>
            <h2 className='text-xl font-medium mb-2'>Host a Server</h2>
            <p className='text-sm text-muted-foreground'>
              Let's get your game server up and running. Please fill out the following form to specify the configuration
              details.
            </p>
          </div>

          <div className='border-t border-border' />

          <form className={'space-y-6'}>
            <div className='space-y-2'>
              <label className='uk-form-label' htmlFor='bindAddress'>Server Address</label>
              <input className='uk-input'
                     type='text'
                     placeholder={'0.0.0.0:2137'}
                     id='bindAddress'
                     name={'bindAddress'}
                     value={form.bindAddress}
                     onChange={handleChange}
              />
              <div className='uk-form-help text-muted-foreground'>
                Enter the IP address & port number to bind the game server to and listen for incoming connections.
                This will be the address that players connect to. Make sure the port is open through any firewalls.
              </div>
            </div>

            <div className='space-y-2'>
              <label className='uk-form-label' htmlFor='databaseType'>Database type</label>
              <select className='uk-select'
                      name='databaseType'
                      id='databaseType'
                      value={form.databaseType}
                      onChange={handleSelect}>
                <option value={'sqlite'}>Saved on disk</option>
                <option value={'memory'}>Stored in-memory</option>
              </select>
              <div className='uk-form-help text-muted-foreground'>
                Select the type of database to use, either SQLite or in-memory.
                Note: in-memory is only recommended for testing.
              </div>
            </div>

            <div className='space-y-2'>
              <label className='uk-form-label' htmlFor='databasePath'>Database path</label>
              <input className='uk-input'
                     type='text'
                     placeholder={'./dispel-multi-db.sqlite'}
                     id='databasePath'
                     name={'databasePath'}
                     value={form.databasePath}
                     disabled={form.databaseType === 'memory'}
                     onChange={handleChange}
              />
              <div className='uk-form-help text-muted-foreground'>
                Enter the path to the database file.
                Note: This field is enabled only when the database is saved on disk.
              </div>
            </div>

            <div className='space-y-2 mt-8'>

              <div className='flex justify-between'>
                {/*<label className='inline-flex items-center gap-x-2 text-xs'*/}
                {/*       htmlFor='mute'> <input*/}
                {/*  className='uk-toggle-switch uk-toggle-switch-primary' id='mute' type='checkbox' />*/}
                {/*  Mute this thread*/}
                {/*</label>*/}


                <Link to={'/'} className='uk-button uk-button-default' uk-toggle='#demo'>
                  <span className='mr-2 w-[16px]' uk-icon='arrow-left' />
                  Go back
                </Link>

                <Link to={'/admin'} className='uk-button uk-button-primary' uk-toggle='#demo'>
                  Host a server
                </Link>
              </div>

              {/*<button className='uk-button uk-button-primary w-full'*/}
              {/*        disabled={true} hidden={false}>*/}
              {/*      <span*/}
              {/*        className='mr-2 uk-icon uk-spinner'*/}
              {/*        uk-spinner='ratio: 0.55' role='status'>*/}
              {/*        <svg width='16.5'*/}
              {/*             height='16.5'*/}
              {/*             viewBox='0 0 30 30'>*/}
              {/*          <circle fill='none' stroke='#000' cx='15' cy='15' r='14'*/}
              {/*                  style={{ strokeWidth: '1.81818px' }} />*/}
              {/*        </svg>*/}
              {/*      </span>*/}
              {/*  Connect*/}
              {/*</button>*/}
            </div>
          </form>

        </div>
      </div>
    </div>
  )
}

export default HostServerFranken