import { Link, NavLink } from 'react-router-dom'

function AdminNavbar() {

  return (
    <nav className='uk-navbar' uk-navbar=''>
      <div className={'uk-navbar-left gap-x-4 lg:gap-x-6'}>
        <div className='flex items-center uk-navbar-item mr-4'>
          <svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='currentColor'
               strokeWidth='2' strokeLinecap='round' strokeLinejoin='round' className='mr-2 h-6 w-6'>
            <path d='M15 6v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3V6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3'></path>
          </svg>
          <h1>Dispel<span className={'font-bold'}>Multi</span></h1>
        </div>
        <ul className='uk-navbar-nav gap-x-4 lg:gap-x-6'>
          <li className={'uk-active'}>
            <NavLink to={'/admin'} className={({ isActive }) => [isActive ? 'uk-active accent-red-300' : "accent-orange-50",].join(" ")} role='button'>Overview</NavLink>
          </li>
          <li><Link to={'/admin/players'} role='button'>Players</Link></li>
          <li><a href='#demo' role='button'>Lobby</a></li>
          <li><a href='#demo' role='button'>Settings</a></li>
        </ul>
      </div>

      <div className='uk-navbar-right gap-x-4 lg:gap-x-6'>
        <div className='uk-navbar-item w-[150px] lg:w-[300px]'>
          <input className='uk-input' placeholder='Search' type='text' disabled={true} />
        </div>
        <div className='uk-navbar-item'>
          <a
            className='inline-flex h-8 w-8 items-center justify-center rounded-full bg-accent ring-ring focus:outline-none focus-visible:ring-1'
            href='#' role='button' aria-haspopup='true'>
          <span
            className='relative flex h-8 w-8 shrink-0 overflow-hidden rounded-full'>
            <span className='inline-block h-full w-full bg-sky-900 text-center text' />
          </span>
          </a>
          <div className='uk-drop uk-dropdown' uk-dropdown='mode: click; pos: bottom-right'>
            <ul className='uk-dropdown-nav uk-nav'>
              <li className='px-2 py-1.5 text-sm'>
                <div className='flex flex-col space-y-1'><p
                  className='text-sm font-medium leading-none'>sveltecult</p> <p
                  className='text-xs leading-none text-muted-foreground'>
                  leader@sveltecult.com
                </p></div>
              </li>
              <li className='uk-nav-divider'></li>
              <li><a className='uk-drop-close justify-between' href='#demo' uk-toggle='' role='button'>
                Profile
                <span className='ml-auto text-xs tracking-widest opacity-60'>⇧⌘P</span> </a></li>
              <li><a className='uk-drop-close justify-between' href='#demo' uk-toggle='' role='button'>
                Billing
                <span className='ml-auto text-xs tracking-widest opacity-60'>⌘B</span> </a></li>
              <li><a className='uk-drop-close justify-between' href='#demo' uk-toggle='' role='button'>
                Settings
                <span className='ml-auto text-xs tracking-widest opacity-60'>⌘S</span> </a></li>
              <li><a className='uk-drop-close justify-between' href='#demo' uk-toggle='' role='button'>
                New Team
              </a></li>
              <li className='uk-nav-divider'></li>
              <li><a className='uk-drop-close justify-between' href='#demo' uk-toggle='' role='button'>
                Logout
              </a></li>
            </ul>
          </div>
        </div>
      </div>
    </nav>
  )
}

export default AdminNavbar