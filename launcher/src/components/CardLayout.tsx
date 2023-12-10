function CardLayout({ children }: {children: any}) {
  return (
    <div className='relative flex min-h-screen flex-col justify-center overflow-hidden bg-gray-700 py-6 sm:py-12'>
      <div
        className='relative bg-gray-800 px-6 pt-10 pb-8 shadow-xl ring-1 ring-gray-100/5 sm:mx-auto sm:max-w-lg sm:rounded-lg sm:px-10'>
        <div className='mx-auto max-w-md'>
          <h1 className={'text-2xl text-gray-50'}>Dispel<span className={'font-bold'}>Multi</span></h1>
          {children}
        </div>
      </div>
      <div className='mt-10'>
        <p className='text-xs text-center text-gray-400'>
          Version 0.0.1 | Build Date: {new Date().toDateString()}
        </p>
      </div>
    </div>
  )
}

export default CardLayout