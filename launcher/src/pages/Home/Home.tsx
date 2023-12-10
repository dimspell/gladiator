function Home() {
  return (
    <div className='relative flex min-h-screen flex-col justify-center overflow-hidden bg-gray-700 py-6 sm:py-12'>
      <div
        className='relative bg-gray-800 px-6 pt-10 pb-8 shadow-xl ring-1 ring-gray-100/5 sm:mx-auto sm:max-w-lg sm:rounded-lg sm:px-10'>
        <div className='mx-auto max-w-md'>
          <h1 className={'text-2xl text-gray-50'}>Dispel<span className={'font-bold'}>Multi</span></h1>
          <div className='divide-y divide-gray-700/50'>
            <div className='space-y-6 py-8 text-base leading-7 text-gray-300'>
              <h2 className='font-bold text-gray-100'>Greetings, brave adventurer!</h2>
              <p className='text-gray-100 leading-'>
                Whether you're stepping into the mystical realms of Dman for the first time or returning for another
                epic journey, we're thrilled to have you here. Prepare yourself for a world of magic, challenges, and
                camaraderie.
              </p>
              <h2 className='font-bold text-gray-100'>Ready to Begin Your Journey?</h2>
              <p className='text-gray-100 leading-'>
                Follow the wizard to host your very own server or choose an existing server to join forces and forge
                alliances as you embark on quests together.
              </p>
              <div className={'flex flex-row space-x-7'}>
                <a href={'#'}
                   className='flex-1 text-center text-sky-100 hover:text-sky-200 bg-sky-700 hover:bg-sky-800 px-5 py-4 rounded'>
                  Join a Server
                </a>
                <a href={'#'}
                   className='flex-1 text-center text-amber-100 hover:text-amber-200 bg-amber-700 hover:bg-amber-800 px-5 py-4 rounded'>
                  Host a Server
                </a>
              </div>
            </div>
            <div className='pt-8 text-base font-semibold leading-7'>
              <p className='text-gray-100'>
                Are you curious about the development?
              </p>
              <p>
                <a href='https://tailwindcss.com/docs' className='text-sky-500 hover:text-sky-600'>
                  Join Discord channel &rarr;
                </a>
              </p>
            </div>
          </div>
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

export default Home