function CardLayout({ children }: {children: any}) {
  return (
    <div className='bg-gray-700 min-h-screen'>
      {children}
      <div className='mt-10'>
        <p className='text-xs text-center text-gray-400'>
          Version 0.0.1 | Build Date: {new Date().toDateString()}
        </p>
      </div>
    </div>
  )
}

export default CardLayout