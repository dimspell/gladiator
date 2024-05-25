import AdminNavbar from '../AdminNavbar'

function AdminPlayersFranken() {

  return (
    <div className='min-h-screen flex flex-col'>
      <div className='border-b border-border px-4'>
        <AdminNavbar />
      </div>

      <div className='flex-1 flex items-center self-center'>
        <div className='uk-card flex min-h-64 p-8 items-center justify-center'>
          <div className='flex flex-1 items-center justify-center gap-x-2 text-destructive'>
            <svg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 24 24' fill='none'
                 stroke='currentColor' strokeWidth='2' strokeLinecap='round' strokeLinejoin='round'
                 className='lucide lucide-circle-alert'>
              <circle cx='12' cy='12' r='10'></circle>
              <line x1='12' x2='12' y1='8' y2='12'></line>
              <line x1='12' x2='12.01' y1='16' y2='16'></line>
            </svg>
            Graph not available
          </div>
        </div>
      </div>
    </div>
  )
}

export default AdminPlayersFranken